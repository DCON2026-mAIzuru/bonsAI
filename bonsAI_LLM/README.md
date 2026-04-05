# bonsAI LLM Node

`llm_document_raspi_only.md` の方針に合わせて、Raspberry Pi 5 (8GB) 上で `llama.cpp` をサーバーモードで動かすための置き場です。

この構成では、LLM ノードは localhost のみで待ち受けます。

- 現在の試験モデル: Gemma 4 E2B Instruct の GGUF 量子化版
- 推奨ポート: `127.0.0.1:8081`
- Go バックエンド接続先: `http://127.0.0.1:8081/v1/chat/completions`
- バックエンドからのモデル名: `gemma-4-e2b-it`

## 前提

- Raspberry Pi OS 64-bit
- Raspberry Pi 5 (8GB)
- 純正 27W 電源
- アクティブクーラー推奨
- できれば NVMe SSD ブート

まず Gemma 4 E2B を試せるようにしていますが、Raspberry Pi 5 (8GB) では重い量子化は厳しい可能性があります。既定は `unsloth/gemma-4-E2B-it-GGUF` の `Q4_K_M` にしてあります。`ctx 2048` は据え置きで、必要に応じてさらに軽くしてください。

## ディレクトリ

- `scripts/bootstrap_llama_cpp.sh`: llama.cpp を取得してビルド
- `scripts/start_llm_server.sh`: 汎用の LLM サーバー起動
- `scripts/start_qwen4b_server.sh`: 旧スクリプト名の互換ラッパー
- `systemd/bonsai-llm.service`: systemd ユニット例
- `.env.example`: Pi 上の LLM 設定テンプレート

## セットアップ

1. 依存を入れる

```bash
sudo apt update
sudo apt install -y git cmake build-essential pkg-config
```

2. `llama.cpp` を取得してビルドする

```bash
cd /path/to/bonsAI/bonsAI_LLM
cp .env.example .env
./scripts/bootstrap_llama_cpp.sh
```

3. Gemma 4 E2B の GGUF を使う

```bash
# 既定の .env.example のままで
# unsloth/gemma-4-E2B-it-GGUF の Q4_K_M を参照します
```

注意:
ローカルの GGUF ファイルを使いたい場合は、`.env` の `BONSAI_LLM_MODEL_FILE` に実ファイルを指定し、`BONSAI_LLM_HF_REPO` と `BONSAI_LLM_HF_FILE` を空にしてください。

既定の Hugging Face 設定は次です。

```bash
BONSAI_LLM_HF_REPO=unsloth/gemma-4-E2B-it-GGUF
BONSAI_LLM_HF_FILE=gemma-4-E2B-it-Q4_K_M.gguf
```

4. サーバーを起動する

```bash
cd /path/to/bonsAI/bonsAI_LLM
./scripts/start_llm_server.sh
```

5. 疎通確認

```bash
curl -s http://127.0.0.1:8081/health
```

## systemd 化

`systemd/bonsai-llm.service` を `/etc/systemd/system/` に置いて調整すると、ラズパイ再起動後も自動起動できます。

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now bonsai-llm.service
sudo systemctl status bonsai-llm.service
```

## バックエンド側の設定

`bonsAI_server/.env.example` の値を使うと、Go バックエンドはこの LLM ノードへ接続します。

```bash
BONSAI_LLM_CHAT_STREAM_URL=http://127.0.0.1:8081/v1/chat/completions
BONSAI_LLM_MODEL=gemma-4-e2b-it
```

## 補足

`llama.cpp` の公式 README では `llama-server -m model.gguf --port 8080` で OpenAI 互換 API を公開できると案内されています。Debian/Ubuntu 系は公式 Wiki でも `git clone` と `cmake` ベースのビルド手順が案内されています。
