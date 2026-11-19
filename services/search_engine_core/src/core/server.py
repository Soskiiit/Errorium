"""gRPC server for search_engine_core."""
from concurrent import futures
import logging
import os
import signal
import sys
import time
from typing import Optional, Tuple

import grpc
import graph_pb2_grpc
import search_pb2_grpc

from .config import CoreConfig
from .engine import Engine
from .graph_servicer import GraphServicer
from .search_servicer import SearchServicer

logging.basicConfig(
    level=logging.INFO,
    format="[%(asctime)s] [%(levelname)s] [%(name)s] %(message)s",
)
logger = logging.getLogger("search_engine_core.server")


def create_server(
    config: Optional[CoreConfig] = None,
    engine: Optional[Engine] = None,
    servicer: Optional[SearchServicer] = None,
    graph_servicer: Optional[GraphServicer] = None,
    max_workers: int = 10,
) -> Tuple[grpc.Server, SearchServicer]:
    """Create and configure gRPC server with registered SearchService and GraphService."""
    cfg = config or CoreConfig.from_env()

    if servicer is None:
        if engine is None:
            try:
                engine = Engine.load(cfg.index_dir, config=cfg)
                logger.info("Loaded search index from %s", cfg.index_dir)
            except Exception as e:
                logger.warning(
                    "Search index could not be loaded on startup (%s); starting in unready state",
                    e,
                )
                engine = None
        servicer = SearchServicer(engine=engine, index_dir=cfg.index_dir, config=cfg)

    if graph_servicer is None:
        graph_servicer = GraphServicer(search_servicer=servicer)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=max_workers))
    search_pb2_grpc.add_SearchServiceServicer_to_server(servicer, server)
    graph_pb2_grpc.add_GraphServiceServicer_to_server(graph_servicer, server)

    return server, servicer


def serve() -> None:
    """Run gRPC server listening on GRPC_PORT with graceful shutdown on SIGTERM/SIGINT."""
    cfg = CoreConfig.from_env()
    server, _ = create_server(config=cfg)

    bind_addr = f"0.0.0.0:{cfg.grpc_port}"
    server.add_insecure_port(bind_addr)
    server.start()
    logger.info("Search Engine Core gRPC server listening on %s", bind_addr)

    stop_event = False

    def handle_shutdown(signum, frame):
        nonlocal stop_event
        logger.info("Received signal %d, initiating graceful shutdown...", signum)
        server.stop(grace=5.0)
        stop_event = True

    signal.signal(signal.SIGINT, handle_shutdown)
    signal.signal(signal.SIGTERM, handle_shutdown)

    try:
        while not stop_event:
            time.sleep(1.0)
    except KeyboardInterrupt:
        logger.info("KeyboardInterrupt received, stopping server...")
        server.stop(grace=5.0)

    logger.info("Server stopped.")


if __name__ == "__main__":
    serve()
