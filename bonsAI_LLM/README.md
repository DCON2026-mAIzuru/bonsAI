# bonsAI LLM Node

`llm_document_raspi_only.md` の方針に合わせて、Raspberry Pi 5 (8GB) 上で `llama.cpp` をサーバーモードで動かすための置き場です。

この構成では、LLM ノードは localhost のみで待ち受けます。

- 推奨モデル: Qwen2.5 3B Instruct の GGUF 4bit 量子化版
- 推奨ポート: `127.0.0.1:8081`
- Go バックエンド接続先: `http://127.0.0.1:8081/v1/chat/completions`
- バックエンドからのモデル名: `qwen2.5-3b`

## 前提

- Raspberry Pi OS 64-bit
- Raspberry Pi 5 (8GB)
- 純正 27W 電源
- アクティブクーラー推奨
- できれば NVMe SSD ブート

設計書の制約に合わせ、メモリ圧迫を避けるため `Qwen2.5 3B + 4bit + ctx 2048` を初期値にしています。

## ディレクトリ

- `scripts/bootstrap_llama_cpp.sh`: llama.cpp を取得してビルド
- `scripts/start_qwen4b_server.sh`: Qwen2.5 3B サーバー起動
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

3. Qwen2.5 3B の GGUF ファイルを配置する

```bash
mkdir -p /opt/bonsAI/models
# 例: /opt/bonsAI/models/qwen2.5-3b-instruct-q4_k_m.gguf
```

注意:
モデルの配布元によって実ファイル名は変わるので、`.env` の `BONSAI_LLM_MODEL_FILE` を実際の GGUF パスに合わせて更新してください。

ローカルファイルを置かずに Hugging Face から直接取得したい場合は、`.env` に次を設定してから起動しても構いません。

```bash
BONSAI_LLM_HF_REPO=Qwen/Qwen2.5-3B-Instruct-GGUF
BONSAI_LLM_HF_FILE=qwen2.5-3b-instruct-q4_k_m.gguf
```

4. サーバーを起動する

```bash
cd /path/to/bonsAI/bonsAI_LLM
./scripts/start_qwen4b_server.sh
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
BONSAI_LLM_MODEL=qwen2.5-3b
```

## 補足

`llama.cpp` の公式 README では `llama-server -m model.gguf --port 8080` で OpenAI 互換 API を公開できると案内されています。Debian/Ubuntu 系は公式 Wiki でも `git clone` と `cmake` ベースのビルド手順が案内されています。
