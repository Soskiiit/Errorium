"""gRPC-серверы SearchService и GraphService (protos/search.proto, protos/graph.proto;
docs/Tech/03-data.md, п. 6.4–6.5). pb2-модули генерируются при сборке образа.
"""
import logging
import os
import time
from concurrent import futures

import grpc

import graph_pb2
import graph_pb2_grpc
import search_pb2
import search_pb2_grpc

from . import __version__
from .config import Config
from .engine import SearchEngine
from .reader import IndexNotReadyError, IndexReader, UnsupportedIndexError

log = logging.getLogger("core.server")


def _hit(node, score: float) -> search_pb2.SearchHit:
    return search_pb2.SearchHit(
        doc_id=node.doc_id, score=score, title=node.title, snippet=node.snippet,
        tags=node.tags, question_score=node.question_score,
        accepted_answer_id=node.accepted_answer_id, best_answer_body=node.best_answer_body,
        best_answer_score=node.best_answer_score, created_at=node.created_at)


class SearchServicer(search_pb2_grpc.SearchServiceServicer):
    def __init__(self, engine: SearchEngine):
        self.engine = engine

    def Search(self, request, context):
        start = time.perf_counter()
        try:
            result = self.engine.search(request.query, request.top_k, request.tags, request.language)
        except (IndexNotReadyError, UnsupportedIndexError) as exc:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, str(exc))
        nodes = self.engine.graph.nodes([d for d, _ in result.hits])
        hits = []
        for doc_id, score in result.hits:
            node = nodes.get(doc_id)
            if node is None:
                continue
            hit = _hit(node, score)
            if request.include_raw_body:
                hit.raw_body = node.snippet
            hits.append(hit)
        return search_pb2.SearchResponse(
            hits=hits, total_hits=result.total_hits,
            latency_ms=(time.perf_counter() - start) * 1000.0,
            tokenized_query=result.tokens)

    def GetDocument(self, request, context):
        node = self.engine.graph.nodes([request.doc_id]).get(request.doc_id)
        if node is None:
            context.abort(grpc.StatusCode.NOT_FOUND, f"документ {request.doc_id} не найден")
        return search_pb2.GetDocumentResponse(document=_hit(node, 0.0))

    def BatchGetDocuments(self, request, context):
        nodes = self.engine.graph.nodes(request.doc_ids)
        documents = [_hit(nodes[d], 0.0) for d in request.doc_ids if d in nodes]
        return search_pb2.BatchGetDocumentsResponse(documents=documents)

    def GetStats(self, request, context):
        reader = self.engine.reader
        try:
            reader.try_load()
        except UnsupportedIndexError as exc:
            log.error("индекс отклонён: %s", exc)
        if not reader.ready:
            return search_pb2.GetStatsResponse(is_ready=False)
        return search_pb2.GetStatsResponse(
            total_documents=reader.n, avg_doc_length=reader.avgdl,
            vocabulary_size=reader.vocabulary_size, index_size_bytes=reader.index_size_bytes,
            is_ready=True, build_timestamp=reader.build_timestamp)

    def HealthCheck(self, request, context):
        reader = self.engine.reader
        try:
            reader.try_load()
        except UnsupportedIndexError:
            pass
        if reader.ready:
            status = search_pb2.HealthCheckResponse.SERVING
        elif os.path.isdir(os.path.join(reader.index_dir, ".building")):
            status = search_pb2.HealthCheckResponse.REBUILDING_INDEX
        else:
            status = search_pb2.HealthCheckResponse.NOT_SERVING
        return search_pb2.HealthCheckResponse(status=status, version=__version__)


class GraphServicer(graph_pb2_grpc.GraphServiceServicer):
    def __init__(self, engine: SearchEngine):
        self.engine = engine

    def GetThreadGraph(self, request, context):
        view = self.engine.graph.thread_graph(request.doc_ids)
        nodes = [graph_pb2.ThreadNode(
            doc_id=n.doc_id,
            category=graph_pb2.NodeCategory.Value("NODE_CATEGORY_" + n.category),
            score=n.score, title=n.title, snippet=n.snippet,
            has_accepted_answer=n.has_accepted_answer, tags=n.tags,
            best_answer_score=n.best_answer_score) for n in view.nodes]
        edges = [graph_pb2.ThreadEdge(
            source_doc_id=e.source, target_doc_id=e.target,
            type=graph_pb2.EdgeType.Value("EDGE_TYPE_" + e.type)) for e in view.edges]
        return graph_pb2.ThreadGraphResponse(nodes=nodes, edges=edges)

    def GetThreadDetails(self, request, context):
        details = self.engine.graph.details(request.doc_id)
        if details is None:
            context.abort(grpc.StatusCode.NOT_FOUND, f"тред {request.doc_id} не найден")
        return graph_pb2.ThreadDetailsResponse(
            doc_id=details.doc_id, title=details.title,
            best_answer_body=details.best_answer_body, original_url=details.original_url)


def build_server(engine: SearchEngine, port: int, max_workers: int = 8):
    """Создаёт и запускает gRPC-сервер; возвращает (server, фактический_порт)."""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=max_workers))
    search_pb2_grpc.add_SearchServiceServicer_to_server(SearchServicer(engine), server)
    graph_pb2_grpc.add_GraphServiceServicer_to_server(GraphServicer(engine), server)
    bound = server.add_insecure_port(f"[::]:{port}")
    if bound == 0:
        raise RuntimeError(f"не удалось занять порт {port}")
    server.start()
    return server, bound


def serve(cfg: Config) -> None:
    reader = IndexReader(cfg.index_dir)
    engine = SearchEngine(reader, k1=cfg.k1, b=cfg.b, max_top_k=cfg.max_top_k)
    try:
        reader.try_load()
    except UnsupportedIndexError as exc:
        log.error("индекс отклонён: %s", exc)
    server, bound = build_server(engine, cfg.port)
    if reader.ready:
        log.info("индекс загружен: N=%d, avgdl=%.1f, словарь=%d термов",
                 reader.n, reader.avgdl, reader.vocabulary_size)
    else:
        log.warning("индекс в %s не опубликован; сервер отвечает NOT_SERVING, "
                    "индекс будет подхвачен при появлении (lazy reload)", cfg.index_dir)
    log.info("Search Engine Core %s слушает gRPC :%d (k1=%.2f, b=%.2f)",
             __version__, bound, cfg.k1, cfg.b)
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(grace=5).wait()
        reader.close()