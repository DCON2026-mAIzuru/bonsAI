# bonsAI Front

Raspberry Pi 向けの軽量な静的Web UIです。`Vite + Preact` で構成しているので、ビルド後は `dist/` をそのまま Go などから静的配信できます。

## 開発

```bash
bun install
bun run dev
```

開発中に `http://localhost:5173` を使う場合、Vite は `/api/*` を既定で `http://127.0.0.1:8082` の Go バックエンドへプロキシします。別ポートで動かすときは `.env` に `BONSAI_DEV_API_TARGET=http://127.0.0.1:XXXX` を設定してください。

## ビルド

```bash
bun run build
```

ビルド済みの静的ファイルは `dist/` に出力されます。

## 想定API

- `GET /api/sensors`
- `POST /api/chat/stream`

`/api/chat/stream` は `text/event-stream` を推奨していますが、通常のテキストストリームでも表示できます。LLM が未接続の場合、フロントはデモ応答に落とさず接続エラーを表示します。

## 設定

開発時は `.env.example` を参考に `VITE_API_BASE` を指定できます。

静的配信後に API の向き先だけ変えたい場合は、`dist/runtime-config.js` の `apiBase` を編集してください。Go バックエンドと同一オリジンで配信する場合は空文字のままで動きます。

## 補足

この作業では `bun` を導入し、`bun install` と `bun run build` まで実行確認済みです。
