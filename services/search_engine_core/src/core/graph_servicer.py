"""gRPC Servicer implementation for GraphService."""
import logging
from typing import Dict, List, Optional, Set

import grpc
import graph_pb2
import graph_pb2_grpc

from .engine import Engine, ThreadStore
from .search_servicer import SearchServicer

logger = logging.getLogger(__name__)


class GraphServicer(graph_pb2_grpc.GraphServiceServicer):
    """Thread-safe gRPC servicer for thread graph exploration and thread details."""

    def __init__(
        self,
        engine: Optional[Engine] = None,
        thread_store: Optional[ThreadStore] = None,
        search_servicer: Optional[SearchServicer] = None,
    ) -> None:
        self._engine = engine
        self._thread_store = thread_store
        self._search_servicer = search_servicer

    @property
    def thread_store(self) -> Optional[ThreadStore]:
        if self._search_servicer is not None:
            eng = self._search_servicer.engine
            return eng.thread_store if eng is not None else None
        if self._engine is not None:
            return self._engine.thread_store
        return self._thread_store

    def GetThreadGraph(
        self,
        request: graph_pb2.ThreadGraphRequest,
        context: grpc.ServicerContext,
    ) -> graph_pb2.ThreadGraphResponse:
        doc_ids = list(request.doc_ids)
        if not doc_ids:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "doc_ids must not be empty")
        if len(doc_ids) > 20:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "doc_ids cannot exceed 20 items")

        store = self.thread_store
        if store is None:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, "index not ready")

        # Deduplicate query doc_ids while preserving input order
        unique_query_ids = list(dict.fromkeys(doc_ids))
        query_nodes_map = store.nodes(unique_query_ids)

        query_nodes: List[graph_pb2.ThreadNode] = []
        retained_query_ids: List[int] = []
        retained_query_ids_set: Set[int] = set()

        is_first = True
        for qid in unique_query_ids:
            node_data = query_nodes_map.get(qid)
            if node_data is None:
                continue

            category = (
                graph_pb2.NODE_CATEGORY_PRIMARY_RESULT
                if is_first
                else graph_pb2.NODE_CATEGORY_OTHER_RESULT
            )
            is_first = False

            retained_query_ids.append(qid)
            retained_query_ids_set.add(qid)
            query_nodes.append(
                graph_pb2.ThreadNode(
                    doc_id=node_data.doc_id,
                    category=category,
                    score=float(node_data.question_score),
                    title=node_data.title,
                    snippet=node_data.snippet,
                    has_accepted_answer=node_data.has_accepted_answer,
                    tags=list(node_data.tags),
                    best_answer_score=node_data.best_answer_score,
                )
            )

        if not retained_query_ids:
            return graph_pb2.ThreadGraphResponse(nodes=[], edges=[])

        # Discover neighbors of query nodes (outgoing links + incoming DUPLICATE_OF links)
        neighbor_edges_raw = store.neighbors(retained_query_ids)

        candidate_weights: Dict[int, float] = {}
        candidate_is_duplicate: Dict[int, bool] = {}

        for edge in neighbor_edges_raw:
            src = edge.source_doc_id
            dst = edge.target_doc_id
            if src == dst:
                # Disallow self-loops
                continue

            neighbor_id: Optional[int] = None
            if src in retained_query_ids_set and dst not in retained_query_ids_set:
                neighbor_id = dst
            elif dst in retained_query_ids_set and src not in retained_query_ids_set:
                neighbor_id = src

            if neighbor_id is None:
                continue

            is_dup = (edge.link_type == "DUPLICATE_OF")
            if neighbor_id in candidate_weights:
                if edge.weight > candidate_weights[neighbor_id]:
                    candidate_weights[neighbor_id] = edge.weight
                if is_dup:
                    candidate_is_duplicate[neighbor_id] = True
            else:
                candidate_weights[neighbor_id] = edge.weight
                candidate_is_duplicate[neighbor_id] = is_dup

        # Filter candidate neighbors to those present in thread_nodes table
        candidate_ids = list(candidate_weights.keys())
        neighbor_nodes_map = store.nodes(candidate_ids) if candidate_ids else {}

        valid_candidate_ids = [cid for cid in candidate_ids if cid in neighbor_nodes_map]
        # Sort candidates: weight DESC, doc_id ASC
        valid_candidate_ids.sort(key=lambda cid: (-candidate_weights[cid], cid))
        selected_neighbor_ids = valid_candidate_ids[:20]

        neighbor_nodes: List[graph_pb2.ThreadNode] = []
        for nid in selected_neighbor_ids:
            node_data = neighbor_nodes_map[nid]
            cat = (
                graph_pb2.NODE_CATEGORY_DUPLICATE
                if candidate_is_duplicate.get(nid, False)
                else graph_pb2.NODE_CATEGORY_LINKED
            )
            neighbor_nodes.append(
                graph_pb2.ThreadNode(
                    doc_id=node_data.doc_id,
                    category=cat,
                    score=float(node_data.question_score),
                    title=node_data.title,
                    snippet=node_data.snippet,
                    has_accepted_answer=node_data.has_accepted_answer,
                    tags=list(node_data.tags),
                    best_answer_score=node_data.best_answer_score,
                )
            )

        all_nodes = query_nodes + neighbor_nodes
        all_node_ids = {n.doc_id for n in all_nodes}

        # Query all edges connecting nodes in all_node_ids
        all_edges_raw = store.neighbors(all_node_ids) if all_node_ids else []
        edges: List[graph_pb2.ThreadEdge] = []
        seen_edges: Set[tuple] = set()

        for edge in all_edges_raw:
            src = edge.source_doc_id
            dst = edge.target_doc_id
            if src == dst:
                continue
            if src in all_node_ids and dst in all_node_ids:
                if edge.link_type == "DUPLICATE_OF":
                    edge_type = graph_pb2.EDGE_TYPE_DUPLICATE_OF
                elif edge.link_type == "LINKED_IN_ANSWER":
                    edge_type = graph_pb2.EDGE_TYPE_LINKED_IN_ANSWER
                else:
                    edge_type = graph_pb2.EDGE_TYPE_UNKNOWN

                edge_key = (src, dst, edge_type)
                if edge_key not in seen_edges:
                    seen_edges.add(edge_key)
                    edges.append(
                        graph_pb2.ThreadEdge(
                            source_doc_id=src,
                            target_doc_id=dst,
                            type=edge_type,
                        )
                    )

        return graph_pb2.ThreadGraphResponse(nodes=all_nodes, edges=edges)

    def GetThreadDetails(
        self,
        request: graph_pb2.ThreadDetailsRequest,
        context: grpc.ServicerContext,
    ) -> graph_pb2.ThreadDetailsResponse:
        store = self.thread_store
        if store is None:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, "index not ready")

        doc_id = request.doc_id
        node = store.details(doc_id)
        if node is None:
            context.abort(grpc.StatusCode.NOT_FOUND, f"Thread {doc_id} not found")

        return graph_pb2.ThreadDetailsResponse(
            doc_id=node.doc_id,
            title=node.title,
            best_answer_body=node.best_answer_body,
            original_url=f"https://stackoverflow.com/questions/{node.doc_id}",
        )
