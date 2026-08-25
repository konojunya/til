# System Design Learning Workspaces

Goでバックエンドの設計パターンを小さく実装し、ローカルの実システムに対するテストで振る舞いを証明するための学習カタログです。

## 共通方針

- 各workspaceは独立したGo moduleとして育てる。
- 作成直後は`README.md`だけを置き、Sectionを始めるまで実装を先回りしない。
- AIが現在の振る舞いを固定するテストを書き、実装・SQL・migration・runtime設定は学習者が入力する。
- PostgreSQL、RedisなどはDockerで起動する。
- AWS境界はAWS SDK for Go v2と[kumo](https://github.com/sivchari/kumo)で再現する。
- API登録、外部アカウント、従量課金がないと成立しないテーマは作らない。
- エミュレータが実物の負荷特性を再現しない場合は、性能値ではなく不変条件・分布・実行計画をテストする。

## Workspace一覧

| Workspace | 状態 | 主題 | ローカル依存 | 優先度 |
| --- | --- | --- | --- | --- |
| [1:1 mutual matching](./one-to-one-matching/) | 進行中 | DB transaction、advisory lock、Transactional Outbox | PostgreSQL、kumo/SQS | 現在 |
| [ranked team matchmaking](./ranked-team-matchmaking/) | 準備済み | 5対5編成、MMR、公平性、待機時間、競合 | PostgreSQL、Redis | 高 |
| [ecommerce recommendation](./ecommerce-recommendation/) | 準備済み | 行動ログ、候補生成、協調フィルタ、ranking | kumo/Kinesis・S3・DynamoDB | 中 |
| [saga compensation](./saga-compensation/) | 準備済み | 分散transaction、補償、再試行、Step Functions | PostgreSQL、kumo/Step Functions | 高 |
| [reliable event consumer](./reliable-event-consumer/) | 準備済み | Inbox、at-least-once、retry、DLQ | PostgreSQL、kumo/SQS | 高 |
| [CQRS order projection](./cqrs-order-projection/) | 準備済み | Write/Read model、非同期projection、rebuild | PostgreSQL、kumo/EventBridge・SQS・DynamoDB | 中 |
| [flash sale inventory](./flash-sale-inventory/) | 準備済み | oversell防止、reservation、lock、期限切れ | PostgreSQL、Redis | 高 |
| [distributed rate limiter](./distributed-rate-limiter/) | 準備済み | token bucket、Redis原子操作、障害方針 | Redis | 中 |
| [cache consistency](./cache-consistency/) | 準備済み | cache-aside、stampede、invalidation、stale read | PostgreSQL、Redis | 高 |
| [partition key design](./partition-key-design/) | 準備済み | hot key、write sharding、順序と分散 | kumo/DynamoDB・Kinesis | 中 |
| [PostgreSQL performance lab](./postgres-performance-lab/) | 準備済み | EXPLAIN、index、pagination、MVCC、lock | PostgreSQL | 高 |
| [event-sourced ledger](./event-sourced-ledger/) | 準備済み | append-only event、楽観lock、snapshot、replay | PostgreSQL、kumo/SQS | 中 |
| [distributed job scheduler](./distributed-job-scheduler/) | 準備済み | lease、heartbeat、fencing token、retry | PostgreSQL | 高 |
| [webhook delivery](./webhook-delivery/) | 準備済み | HMAC署名、delivery log、retry、replay | PostgreSQL、local HTTP receiver | 高 |
| [resumable file processing](./resumable-file-processing/) | 準備済み | presigned upload、multipart、checksum、非同期処理 | PostgreSQL、kumo/S3・SQS | 高 |
| [online schema migration](./online-schema-migration/) | 準備済み | expand/contract、dual write、backfill、無停止DDL | PostgreSQL | 高 |
| [multi-tenant data isolation](./multi-tenant-data-isolation/) | 準備済み | composite key、RLS、pool安全性、tenant quota | PostgreSQL、Redis | 高 |
| [resilient HTTP client](./resilient-http-client/) | 準備済み | timeout budget、retry、circuit breaker、bulkhead | local fault-injection HTTP server | 高 |
| [PostgreSQL search/autocomplete](./postgres-search-autocomplete/) | 準備済み | full-text、trigram、ranking、stable pagination | PostgreSQL | 中 |
| [CDC projection pipeline](./cdc-projection-pipeline/) | 準備済み | logical replication、LSN、snapshot、projection | PostgreSQL、kumo/Kinesis・DynamoDB | 中 |
| [game leaderboard](./game-leaderboard/) | 準備済み | Sorted Set、同点順位、season、rebuild | PostgreSQL、Redis | 中 |
| [WebSocket presence](./websocket-presence/) | 準備済み | heartbeat、multi-device、fan-out、offline resume | PostgreSQL、Redis | 中 |
| [feature flag service](./feature-flag-service/) | 準備済み | stable rollout、override、cache、kill switch | kumo/DynamoDB・EventBridge・SQS | 中 |
| [tamper-evident audit log](./tamper-evident-audit-log/) | 準備済み | append-only、hash chain、署名checkpoint、redaction | PostgreSQL、kumo/S3 | 中 |
| [ML preprocessing pipeline](./ml-preprocessing-pipeline/) | 準備済み | streaming I/O、chunk、checkpoint、quality、lineage | PostgreSQL、kumo/S3・SQS | 高 |
| [RAG document lifecycle](./rag-document-lifecycle/) | 準備済み | chunking、hybrid search、ACL、更新/削除、citation | PostgreSQL/pgvector、kumo/S3・SQS | 高 |
| [LLM evaluation release gate](./llm-evaluation-release-gate/) | 準備済み | regression dataset、grader、人手校正、release gate | PostgreSQL、local model fixtures | 高 |
| [safe agent tool execution](./safe-agent-tool-execution/) | 準備済み | validation、authorization、approval、idempotency、budget | PostgreSQL、kumo/SQS、local tool servers | 高 |
| [LLM request gateway](./llm-request-gateway/) | 準備済み | capability routing、rate limit、fallback、degradation | PostgreSQL、Redis、local provider servers | 高 |
| [AI observability and cost attribution](./ai-observability-cost-attribution/) | 準備済み | trace、TTFT、quality、usage ledger、cost attribution | PostgreSQL、OpenTelemetry | 高 |
| [LLM low-latency response](./llm-low-latency-response/) | 準備済み | prompt/exact/semantic cache、streaming、false-hit防止 | PostgreSQL/pgvector、Redis、local LLM server | 高 |

## おすすめのつながり

1. `one-to-one-matching`のOutbox publisherを完了する。
2. `reliable-event-consumer`でOutboxの反対側にある重複受信とInboxを学ぶ。
3. `saga-compensation`で複数サービスにまたがる失敗と補償へ進む。
4. `postgres-performance-lab`、`cache-consistency`、`flash-sale-inventory`でDBと高負荷時の挙動を深める。
5. `distributed-job-scheduler`と`resilient-http-client`でbackground処理と外部通信の共通基盤を作る。
6. `webhook-delivery`と`resumable-file-processing`で信頼できない外部境界とobject処理を学ぶ。
7. `online-schema-migration`と`multi-tenant-data-isolation`で稼働中DBの変更とdata securityを学ぶ。
8. `cqrs-order-projection`、`event-sourced-ledger`、`cdc-projection-pipeline`でread modelを作る3つの方法を比較する。
9. `postgres-search-autocomplete`、`game-leaderboard`、`websocket-presence`でproduct機能向けのread/query modelを作る。
10. `feature-flag-service`と`tamper-evident-audit-log`で安全なreleaseと運用証跡を扱う。
11. `ranked-team-matchmaking`、`ecommerce-recommendation`、`partition-key-design`でアルゴリズムと分散を組み合わせる。
12. `ml-preprocessing-pipeline`で大量dataを一定memoryかつ再実行可能にAIへ渡す入口を作る。
13. `rag-document-lifecycle`で更新・削除・ACLを守る検索基盤を作り、`llm-evaluation-release-gate`で変更による品質低下を止める。
14. `llm-request-gateway`でprovider障害とquotaを制御し、`llm-low-latency-response`でprompt cacheとresponse cache、streamingを分けて速度を改善する。
15. `safe-agent-tool-execution`で副作用を安全に扱い、`ai-observability-cost-attribution`で1 requestの失敗・品質・費用を追跡する。

これは固定順ではありません。各READMEが独立した入口なので、気になったworkspaceのSection 1から始められます。

## 今回workspaceを作らなかったテーマ

### Private live video streaming

保留します。署名付きURL、短命token、HLS playlist・segmentの認可、CDN cache policyだけならS3・CloudFront・KMS相当で学べます。一方、実運用のlive ingest、transcode、ABR ladder、packaging、origin failoverはAWS Elemental MediaLive/MediaPackageやAmazon IVSなどが中心で、現在のkumoの対応範囲だけではend-to-endに再現できません。そこをGoとffmpegだけで置き換えると「private配信」より「自作media pipeline」の学習になるため、今は雛形を作りません。

### Aurora固有のチューニング

保留します。SQL、index、lock、MVCCなどPostgreSQLと共通する部分は`postgres-performance-lab`で扱いますが、Auroraの分散storage、replica lag、I/O課金、failover特性は通常のDocker PostgreSQLでは証明できません。

### Spanner固有のhot spot対策

AWS中心という今回の範囲からは外し、移植可能な「単調増加key、tenant hot key、shard suffix、fan-in read、順序と分散のtrade-off」を`partition-key-design`でDynamoDB/Kinesis上に再構成します。kumoは実AWSのcapacity throttlingを再現しないため、partition分布を計測する決定論的テストを証拠にします。

候補を追加するときも、外部登録なしで最終挙動をテストできるかを先に判定します。
