# TIL

**Today I Learned.**

AI Codingによって、知らない技術でも短時間で動くものを作れるようになりました。一方で、生成されたcodeをそのまま受け取るだけでは、設計判断、失敗時の挙動、transactionや並行処理の境界まで自分の知識として残らないことがあります。

このrepositoryは、AI Coding時代にsystemを深く学ぶため、あえてproduction codeを手で書く学習projectです。AIには全体設計、test、reference code、背景の説明を補助してもらい、人間が実装を入力して、その意味を理解しながら完成させます。最初は全く知らないsystemでも、小さなSectionへ分け、疑問を解消し、最後にtestで振る舞いを証明できるところまで進めます。

## 学習の考え方

- 最初にsystem全体像、外部system、全Section、最終的に証明する振る舞いを確認する。
- 一度に扱うのは1つのSection、1つの振る舞いだけにする。
- AIが「何を守るtestか」「なぜ必要か」を説明してtestを書き、意図したRedを作る。
- AIが最小限のreference codeと設計理由を示し、人間がproduction code、SQL、migration、runtime設定を手で入力する。
- Greenになったら実装とtest evidenceを振り返り、次のSectionへ進む。
- 最後はunit testだけでなく、必要に応じてintegration、concurrency、failure、end-to-end testでsystemを再現する。

手で書くこと自体が目的ではありません。AIへ丸ごと委ねず、設計と実装の因果関係を自分で説明できる状態にすることが目的です。

## 技術方針

- 実装言語は基本的にGoを使う。
- PostgreSQL、RedisなどはDockerで起動し、localで再現する。
- AWS境界はAWS SDK for Go v2と[kumo](https://github.com/sivchari/kumo)を使ってtestする。
- API登録や有料の外部serviceが必須になる題材は避け、localで完結する構成を選ぶ。
- workspaceは最初にREADMEだけを作り、実装を先回りせずSectionごとに育てる。

より詳しい共通方針、学習順、保留中の題材は[System Design Learning Workspaces](./systems/README.md)にまとめています。

## Systems

`進行中`は現在実装しているworkspace、`準備済み`は全体設計とSection roadmapまで用意されているworkspaceです。各linkから個別の全体設計、学習内容、最終acceptanceを確認できます。

| Category | System | 状態 | 作るもの / 学べること |
| --- | --- | --- | --- |
| Matching / Product | [1:1 Mutual Matching](./systems/one-to-one-matching/README.md) | 進行中 | 相互Likeから1対1のmatchを作り、transaction、advisory lock、Transactional Outboxまで実装する。 |
| Matching / Product | [Ranked Team Matchmaking](./systems/ranked-team-matchmaking/README.md) | 準備済み | MMR、待機時間、公平性を考慮してrank制の5対5teamを編成する。 |
| Matching / Product | [E-commerce Recommendation](./systems/ecommerce-recommendation/README.md) | 準備済み | 行動logから候補生成、協調filtering、rankingを行うrecommendation pipelineを作る。 |
| Matching / Product | [Flash Sale Inventory](./systems/flash-sale-inventory/README.md) | 準備済み | 高並行なsaleでoversellを防ぐ在庫reservationと期限切れ処理を作る。 |
| Matching / Product | [Game Leaderboard](./systems/game-leaderboard/README.md) | 準備済み | Redis Sorted Setを使い、同点順位、season、永続化からのrebuildを扱う。 |
| Matching / Product | [PostgreSQL Search / Autocomplete](./systems/postgres-search-autocomplete/README.md) | 準備済み | PostgreSQLのfull-text searchとtrigramで検索、補完、ranking、paginationを作る。 |
| Matching / Product | [WebSocket Presence](./systems/websocket-presence/README.md) | 準備済み | heartbeat、multi-device、fan-out、再接続を含むonline presenceを作る。 |
| Matching / Product | [Feature Flag Service](./systems/feature-flag-service/README.md) | 準備済み | stable rollout、個別override、cache、kill switchを持つfeature flag基盤を作る。 |
| Data / Event | [Saga Compensation](./systems/saga-compensation/README.md) | 準備済み | 複数serviceにまたがる処理をSagaと補償transactionで安全に戻す。 |
| Data / Event | [Reliable Event Consumer](./systems/reliable-event-consumer/README.md) | 準備済み | Inbox、idempotency、retry、DLQでat-least-once messageを安全に処理する。 |
| Data / Event | [CQRS Order Projection](./systems/cqrs-order-projection/README.md) | 準備済み | command側とread modelを分離し、非同期projectionとrebuildを実装する。 |
| Data / Event | [Event-sourced Ledger](./systems/event-sourced-ledger/README.md) | 準備済み | append-only event、楽観lock、snapshot、replayでledgerを再構築する。 |
| Data / Event | [CDC Projection Pipeline](./systems/cdc-projection-pipeline/README.md) | 準備済み | PostgreSQL logical replicationのLSNとsnapshotからread modelを作る。 |
| Data / Event | [Partition Key Design](./systems/partition-key-design/README.md) | 準備済み | hot keyを避けるwrite shardingと、順序・分散・fan-in readのtrade-offを検証する。 |
| Database / Consistency | [Cache Consistency](./systems/cache-consistency/README.md) | 準備済み | cache-aside、invalidation、stampede、stale readの整合性を扱う。 |
| Database / Consistency | [PostgreSQL Performance Lab](./systems/postgres-performance-lab/README.md) | 準備済み | EXPLAIN、index、pagination、MVCC、lockを実測してqueryを改善する。 |
| Database / Consistency | [Online Schema Migration](./systems/online-schema-migration/README.md) | 準備済み | expand/contract、dual write、backfillで稼働中schemaを安全に変更する。 |
| Database / Consistency | [Multi-tenant Data Isolation](./systems/multi-tenant-data-isolation/README.md) | 準備済み | composite key、Row Level Security、connection poolでtenant dataを隔離する。 |
| Reliability / Operations | [Distributed Job Scheduler](./systems/distributed-job-scheduler/README.md) | 準備済み | lease、heartbeat、fencing token、retryを持つ分散job schedulerを作る。 |
| Reliability / Operations | [Distributed Rate Limiter](./systems/distributed-rate-limiter/README.md) | 準備済み | Redisのatomic operationで分散token bucketと障害時policyを実装する。 |
| Reliability / Operations | [Resilient HTTP Client](./systems/resilient-http-client/README.md) | 準備済み | timeout budget、retry、circuit breaker、bulkheadを持つHTTP clientを作る。 |
| Reliability / Operations | [Webhook Delivery](./systems/webhook-delivery/README.md) | 準備済み | HMAC署名、delivery log、retry、replayを備えたwebhook配信基盤を作る。 |
| Reliability / Operations | [Resumable File Processing](./systems/resumable-file-processing/README.md) | 準備済み | multipart upload、checksum、非同期処理、失敗後の再開を実装する。 |
| Reliability / Operations | [Tamper-evident Audit Log](./systems/tamper-evident-audit-log/README.md) | 準備済み | append-only log、hash chain、署名checkpointで改ざんを検知する。 |
| AI / ML Infrastructure | [ML Preprocessing Pipeline](./systems/ml-preprocessing-pipeline/README.md) | 準備済み | 巨大fileを一定memoryで変換し、checkpoint、quality、lineageを残す。 |
| AI / ML Infrastructure | [RAG Document Lifecycle](./systems/rag-document-lifecycle/README.md) | 準備済み | 文書のchunking、hybrid search、ACL、更新・削除、citationを扱う。 |
| AI / ML Infrastructure | [LLM Evaluation Release Gate](./systems/llm-evaluation-release-gate/README.md) | 準備済み | prompt、model、retrieval、tool変更による品質低下を評価してreleaseを止める。 |
| AI / ML Infrastructure | [Safe Agent Tool Execution](./systems/safe-agent-tool-execution/README.md) | 準備済み | validation、権限、承認、idempotency、budgetで副作用のあるtoolを安全に実行する。 |
| AI / ML Infrastructure | [LLM Request Gateway](./systems/llm-request-gateway/README.md) | 準備済み | capability routing、rate limit、fallback、degraded modeを持つgatewayを作る。 |
| AI / ML Infrastructure | [AI Observability and Cost Attribution](./systems/ai-observability-cost-attribution/README.md) | 準備済み | 1 requestのtrace、品質、token、機能・顧客・処理段階別costを追跡する。 |
| AI / ML Infrastructure | [LLM Low-latency Response](./systems/llm-low-latency-response/README.md) | 準備済み | prompt/exact/semantic cache、streaming、TTFT、false-hit防止を比較・実装する。 |

## 始め方

1. 上の一覧から学びたいsystemを選び、READMEの全体像とSection roadmapを読む。
2. AIへ「Section 1を始めたい」と伝え、全Sectionと最終成果の説明を受ける。
3. testが守る振る舞いと必要性を理解してから、1つ目のRedを作る。
4. reference codeと説明を手がかりにproduction codeを自分で入力する。
5. testをGreenにし、全Sectionが終わるまで小さく繰り返す。
