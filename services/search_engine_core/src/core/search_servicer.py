"""gRPC Servicer implementation for SearchService."""
import logging
import threading
from typing import List, Optional

import grpc
import search_pb2
import search_pb2_grpc

from .config import CoreConfig
from .engine import Engine, ThreadNode

logger = logging.getLogger(__name__)


def _node_to_search_hit(node: ThreadNode, score: float = 0.0, raw_body: str = "") -> search_pb2.SearchHit:
    return search_pb2.SearchHit(
        doc_id=node.doc_id,
        score=score,
        title=node.title,
        snippet=node.snippet,
        raw_body=raw_body,
        tags=list(node.tags),
        question_score=node.question_score,
        accepted_answer_id=node.accepted_answer_id,
        best_answer_body=node.best_answer_body,
        best_answer_score=node.best_answer_score,
        created_at=node.created_at,
    )


class SearchServicer(search_pb2_grpc.SearchServiceServicer):
    """Thread-safe gRPC servicer wrapping the BM25 search engine and thread store."""

    def __init__(
        self,
        engine: Optional[Engine] = None,
        index_dir: Optional[str] = None,
        config: Optional[CoreConfig] = None,
    ) -> None:
        self._lock = threading.Lock()
        self._engine: Optional[Engine] = engine
        self._config = config or (engine.config if engine else CoreConfig.from_env())
        self._index_dir = index_dir or self._config.index_dir

    @property
    def engine(self) -> Optional[Engine]:
        with self._lock:
            return self._engine

    def Search(self, request: search_pb2.SearchRequest, context: grpc.ServicerContext) -> search_pb2.SearchResponse:
        # Validate query
        query = request.query.strip() if request.query else ""
        if not query:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "query must not be empty")

        current_engine = self.engine
        if current_engine is None:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, "index not ready")

        top_k = request.top_k if request.top_k > 0 else 10
        tags = list(request.tags) if request.tags else None
        language = request.language.strip() if request.language else None

        result = current_engine.search(
            query=request.query,
            top_k=top_k,
            tags=tags,
            language=language,
        )

        hits: List[search_pb2.SearchHit] = []
        for h in result.hits:
            # Full question body is not stored in index v1 format;
            # therefore raw_body is populated with snippet when include_raw_body=True.
            raw_body = h.snippet if request.include_raw_body else ""
            hits.append(
                search_pb2.SearchHit(
                    doc_id=h.doc_id,
                    score=h.score,
                    title=h.title,
                    snippet=h.snippet,
                    raw_body=raw_body,
                    tags=list(h.tags),
                    question_score=h.question_score,
                    accepted_answer_id=h.accepted_answer_id,
                    best_answer_body=h.best_answer_body,
                    best_answer_score=h.best_answer_score,
                    created_at=h.created_at,
                )
            )

        return search_pb2.SearchResponse(
            hits=hits,
            total_hits=result.total_hits,
            latency_ms=result.latency_ms,
            tokenized_query=result.tokenized_query,
        )

    def GetDocument(self, request: search_pb2.GetDocumentRequest, context: grpc.ServicerContext) -> search_pb2.GetDocumentResponse:
        current_engine = self.engine
        if current_engine is None:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, "index not ready")

        node = current_engine.thread_store.details(request.doc_id)
        if node is None:
            context.abort(grpc.StatusCode.NOT_FOUND, f"Document {request.doc_id} not found")

        hit = _node_to_search_hit(node, score=0.0, raw_body=node.snippet)
        return search_pb2.GetDocumentResponse(document=hit)

    def BatchGetDocuments(self, request: search_pb2.BatchGetDocumentsRequest, context: grpc.ServicerContext) -> search_pb2.BatchGetDocumentsResponse:
        current_engine = self.engine
        if current_engine is None:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, "index not ready")

        doc_ids = list(request.doc_ids)
        if not doc_ids:
            return search_pb2.BatchGetDocumentsResponse(documents=[])

        nodes_map = current_engine.thread_store.nodes(doc_ids)
        documents: List[search_pb2.SearchHit] = []
        for doc_id in doc_ids:
            node = nodes_map.get(doc_id)
            if node is not None:
                documents.append(_node_to_search_hit(node, score=0.0, raw_body=node.snippet))

        return search_pb2.BatchGetDocumentsResponse(documents=documents)

    def GetStats(self, request: search_pb2.GetStatsRequest, context: grpc.ServicerContext) -> search_pb2.GetStatsResponse:
        current_engine = self.engine
        if current_engine is None:
            return search_pb2.GetStatsResponse(
                total_documents=0,
                avg_doc_length=0.0,
                vocabulary_size=0,
                index_size_bytes=0,
                is_ready=False,
                build_timestamp=0,
            )

        stats = current_engine.stats()
        return search_pb2.GetStatsResponse(
            total_documents=stats.total_documents,
            avg_doc_length=stats.avg_doc_length,
            vocabulary_size=stats.vocabulary_size,
            index_size_bytes=stats.index_size_bytes,
            is_ready=stats.is_ready,
            build_timestamp=stats.build_timestamp,
        )

    def ReloadIndex(self, request: search_pb2.ReloadIndexRequest, context: grpc.ServicerContext) -> search_pb2.ReloadIndexResponse:
        try:
            new_engine = Engine.load(self._index_dir, config=self._config)
            build_ts = new_engine.stats().build_timestamp
            with self._lock:
                self._engine = new_engine
            logger.info("Successfully reloaded index from %s (build_ts=%d)", self._index_dir, build_ts)
            return search_pb2.ReloadIndexResponse(ok=True, message="", build_timestamp=build_ts)
        except Exception as e:
            logger.warning("Failed to reload index from %s: %s", self._index_dir, e)
            current_engine = self.engine
            old_ts = current_engine.stats().build_timestamp if current_engine else 0
            return search_pb2.ReloadIndexResponse(ok=False, message=str(e), build_timestamp=old_ts)

    def HealthCheck(self, request: search_pb2.HealthCheckRequest, context: grpc.ServicerContext) -> search_pb2.HealthCheckResponse:
        current_engine = self.engine
        if current_engine is not None and current_engine.stats().is_ready:
            status = search_pb2.HealthCheckResponse.ServingStatus.SERVING
        else:
            status = search_pb2.HealthCheckResponse.ServingStatus.NOT_SERVING
        return search_pb2.HealthCheckResponse(status=status, version="1.0.0")
