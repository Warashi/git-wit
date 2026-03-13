# AGENTS.md

このファイルは `/home/warashi/ghq/github.com/Warashi/git-wit` 向けのプロジェクト固有ルールのみを扱う。ユーザー単位や Codex 全体の共通ルールはここに再掲しない。

## プロジェクト概要

- `git-wit` は Git Worktree を ID とメタデータで管理する個人向け CLI ツールである。
- 現時点の設計上の一次情報は [`docs/design.md`](docs/design.md) とし、コマンド仕様、状態モデル、同期ポリシーの詳細はそちらを参照する。
- `AGENTS.md` は設計書の要約ではなく、実装や運用で常に意識すべき判断基準だけを保持する。必要になったらこのファイルもメンテナンスすること。

## 実装方針

- このプロジェクトでは破壊的変更を前提として最適化を考えてよい。後方互換より、設計の一貫性と将来の実装速度を優先する。
- パッケージ構成は `package by feature` を原則とする。`add`、`ls`、`rm`、`merge`、`prune` などのユースケース単位で責務を閉じ、Git 操作やメタデータ操作も feature を起点に整理する。
- ただし feature をまたぐ純粋な基盤処理だけは共有パッケージに切り出してよい。共有化は重複除去よりも依存関係の明確化を優先して判断する。
- 実装は [`docs/design.md`](docs/design.md) の状態空間と CLI 契約に整合していること。とくに `refs/git-wit/<ID>`、JSON メタデータ、`ignored` / `untracked` の同期ルールを勝手に簡略化しない。

## 品質ゲート

- Go プロジェクトとして `go test ./...` を通すこと。
- lint は `golangci-lint run` を基準とし、設定は [`.golangci.yaml`](.golangci.yaml) に従う。
- Nix 開発環境では `nix flake check` に `go-test` と `go-lint` のチェックが定義されている。環境差異が気になる場合はこれを優先して確認する。

## ドキュメント運用

- `AGENTS.md` を含むドキュメント群は MECE を保つこと。設計、実装方針、作業手順を同じ粒度で重複記載しない。
- 重複する説明が必要な場合は本文を複製せず、一次情報へのリンクで済ませること。設計の詳細は [`docs/design.md`](docs/design.md) を参照先にする。
- 実装が進んでこのファイルの前提が古くなったら、コード変更に合わせて `AGENTS.md` も更新すること。
