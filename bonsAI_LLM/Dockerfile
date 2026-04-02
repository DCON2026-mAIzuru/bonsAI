FROM debian:bookworm-slim AS builder

ARG LLAMA_CPP_REF=b5680

WORKDIR /opt

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    cmake \
    curl \
    git \
    pkg-config \
  && rm -rf /var/lib/apt/lists/*

RUN git clone --depth=1 --branch "${LLAMA_CPP_REF}" https://github.com/ggml-org/llama.cpp.git /opt/llama.cpp
RUN cmake -S /opt/llama.cpp -B /opt/llama.cpp/build -DCMAKE_BUILD_TYPE=Release
RUN cmake --build /opt/llama.cpp/build --config Release -j4

FROM debian:bookworm-slim

WORKDIR /opt

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    bash \
    ca-certificates \
    curl \
    libgomp1 \
    libstdc++6 \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/bonsAI/bonsAI_LLM

COPY bonsAI_LLM/scripts ./scripts
COPY --from=builder /opt/llama.cpp/build/bin/llama-server /opt/llama.cpp/build/bin/llama-server
RUN chmod +x ./scripts/bootstrap_llama_cpp.sh ./scripts/start_qwen4b_server.sh

ENV BONSAI_LLM_SERVER_BIN=/opt/llama.cpp/build/bin/llama-server
ENV BONSAI_LLM_HOST=0.0.0.0
ENV BONSAI_LLM_PORT=8081
ENV BONSAI_LLM_ALIAS=qwen2.5-3b
ENV BONSAI_LLM_CTX_SIZE=2048
ENV BONSAI_LLM_THREADS=4
ENV BONSAI_LLM_PARALLEL=1
ENV BONSAI_LLM_BATCH=256
ENV BONSAI_LLM_UBATCH=128
ENV BONSAI_LLM_GPU_LAYERS=0

EXPOSE 8081

CMD ["./scripts/start_qwen4b_server.sh"]
