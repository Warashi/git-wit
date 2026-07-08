# AGENTS.md

このファイルは `git-wit` 向けのプロジェクト固有ルールのみを扱う。ユーザー単位や全体の共通ルールはここに再掲しない。

## プロジェクト概要

- `git-wit` は Git Worktree を ID とメタデータで管理する個人向け CLI ツールである。
- 高レベルの仕様と状態モデルの一次情報は [`docs/design.md`](docs/design.md) とし、package 境界と局所 invariant は `internal/wit/*/doc.go` を参照する。
- `AGENTS.md` は設計書の要約ではなく、実装や運用で常に意識すべき判断基準だけを保持する。必要になったらこのファイルもメンテナンスすること。

## 実装方針

- このプロジェクトでは破壊的変更を前提として最適化を考えてよい。後方互換より、設計の一貫性と将来の実装速度を優先する。
- Richard Gabriel の LoB （Locality of Behavior） を意識して、コードとドキュメントは関連する内容を近くに置くことを心がける。
- パッケージ構成は state transition を主軸にした `package by feature` を原則とする。作成は `internal/wit/create`、参照は `internal/wit/query`、統合と削除は `internal/wit/integrate`、整合性回復は `internal/wit/reconcile` に閉じる。
- feature をまたぐ共有処理は `internal/wit/catalog` と `internal/wit/sync` に限定し、さらに下に共有層を増やさない。純粋な platform 処理だけは `internal/git` に置いてよい。
- 実装は [`docs/design.md`](docs/design.md) の状態空間と CLI 契約に整合していること。とくに `refs/git-wit/<ID>`、JSON メタデータ、`ignored` / `untracked` の同期ルールを勝手に簡略化しない。

## 品質ゲート

- Go プロジェクトとして `go test ./...` を通すこと。
- lint は `golangci-lint run` を基準とし、設定は [`.golangci.yaml`](.golangci.yaml) に従う。
- Nix 開発環境では `nix flake check` に `go-test` と `go-lint` のチェックが定義されている。環境差異が気になる場合はこれを優先して確認する。

## ドキュメント運用

- `AGENTS.md` を含むドキュメント群は MECE を保つこと。設計、実装方針、作業手順を同じ粒度で重複記載しない。
- 重複する説明が必要な場合は本文を複製せず、一次情報へのリンクで済ませること。高レベル仕様は [`docs/design.md`](docs/design.md)、package ローカルな責務は `internal/wit/*/doc.go` を参照先にする。
- 実装が進んでこのファイルの前提が古くなったら、コード変更に合わせて `AGENTS.md` も更新すること。
