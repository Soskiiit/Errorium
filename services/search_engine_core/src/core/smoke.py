"""Smoke test client for search_engine_core gRPC service."""
import os
import sys

import grpc
import search_pb2
import search_pb2_grpc


def run_smoke(addr: str) -> bool:
    """Run HealthCheck, Search, and GetStats against gRPC server. Returns True on success."""
    print(f"==> Connecting to Search Engine Core at {addr}...")
    try:
        with grpc.insecure_channel(addr) as channel:
            stub = search_pb2_grpc.SearchServiceStub(channel)

            # 1. HealthCheck
            print("--> 1. HealthCheck...")
            health = stub.HealthCheck(search_pb2.HealthCheckRequest(), timeout=5.0)
            status_name = search_pb2.HealthCheckResponse.ServingStatus.Name(health.status)
            print(f"    Status: {status_name}, Version: {health.version}")
            if health.status != search_pb2.HealthCheckResponse.ServingStatus.SERVING:
                print(f"    WARNING: Health status is not SERVING ({status_name})")

            # 2. Search
            query = "python IndexError"
            print(f"--> 2. Search(query={query!r}, top_k=10)...")
            search_res = stub.Search(
                search_pb2.SearchRequest(query=query, top_k=10, include_raw_body=False),
                timeout=5.0,
            )
            print(f"    Total hits: {search_res.total_hits}, Latency: {search_res.latency_ms:.2f} ms")
            print(f"    Tokenized query: {list(search_res.tokenized_query)}")
            print(f"    Returned hits count: {len(search_res.hits)}")
            for idx, hit in enumerate(search_res.hits[:3]):
                print(f"    Hit [{idx + 1}]: doc_id={hit.doc_id}, score={hit.score:.4f}, title={hit.title!r}")

            # 3. GetStats
            print("--> 3. GetStats...")
            stats = stub.GetStats(search_pb2.GetStatsRequest(), timeout=5.0)
            print(f"    Total documents (N): {stats.total_documents}")
            print(f"    Average doc length (avgdl): {stats.avg_doc_length:.2f}")
            print(f"    Vocabulary size: {stats.vocabulary_size}")
            print(f"    Index size bytes: {stats.index_size_bytes}")
            print(f"    Is ready: {stats.is_ready}")
            print(f"    Build timestamp: {stats.build_timestamp}")

            print("==> Smoke test passed successfully!")
            return True
    except Exception as e:
        print(f"==> Smoke test FAILED: {e}", file=sys.stderr)
        return False


def main() -> None:
    addr = "127.0.0.1:50051"
    if len(sys.argv) > 1:
        addr = sys.argv[1]
    elif "GRPC_ADDR" in os.environ:
        addr = os.environ["GRPC_ADDR"]

    success = run_smoke(addr)
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
