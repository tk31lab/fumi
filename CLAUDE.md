# CLAUDE.md

fumi は、ポートを公開していない Docker ネットワーク内のサービス（Compose の DB など）に届くための Go 製 CLI です。
利用者向けの説明は README.md にあります。

## 開発コマンド

```sh
go build ./...
go vet ./...
go test ./...
gofmt -l .                                   # 何も出なければ OK
GOOS=windows go vet ./... && GOOS=darwin go build -o /dev/null .   # 他 OS 向けのビルド確認
```

- Go のバージョンは `go.mod` の `go 1.27.0`。手元の Go が古くても、ツールチェーンが自動でダウンロードされる。
- Docker SDK は `github.com/moby/moby/client` と `github.com/moby/moby/api` を使う。`github.com/docker/docker` は v28 で更新が止まっているので使わない。

## 構成

```
main.go                cmd.Execute() を呼ぶだけ
cmd/                   cobra のコマンド定義（run / forward / ls / rm / networks）。引数の処理と表示だけを書く
internal/bastion/      fumi run：コンテナの作成・attach・raw モード・リサイズ・/work のマウント
internal/forward/      fumi forward：引数（[LOCAL_PORT:]HOST:PORT）の解析、socat コンテナ
internal/managed/      fumi ls / rm：fumi が起動したコンテナの一覧・名前の解決・削除
internal/network/      ネットワークの一覧・存在確認・ネットワーク上の DNS 名
internal/dockerutil/   Docker への接続、イメージの pull、名前が重複したら作り直すコンテナ作成、ラベル
internal/namegen/      日本庭園テーマのコンテナ名（fumi-<ようす>-<庭にあるもの>）
```

複数のコマンドで使う処理は `dockerutil` / `network` に置き、コマンドごとに書き直さない。

## 守っている決まり

- **ラベル**：fumi が作るコンテナには必ず `dockerutil.Labels(kind)` を付ける（`fumi.managed=true` と `fumi.kind=shell|forward`）。forward には転送内容の `fumi.forward` も付ける。`fumi ls` / `fumi rm` はこのラベルで見分ける。
- **fumi 以外のコンテナには触らない**：`fumi rm` は `fumi.managed=true` のコンテナしか削除しない。
- **後片付け**：コンテナは `AutoRemove` で作り、SIGTERM / SIGHUP（forward は SIGINT も）を受けたら強制削除する。コンテナを残したまま終わる経路を作らない。
- **forward は 127.0.0.1 だけに公開する**：セキュリティ上の決定。ほかのアドレスで公開するオプションは追加しない。
- **イメージはバージョンを固定する**（`alpine:3.24`、`alpine/socat:1.8.1.3`）。`latest` は使わない。
- **補完の関数**は Tab を押すたびに実行されるので、速く、副作用がなく、エラー時も何も出力しない（`ShellCompDirectiveNoFileComp` を返すだけ）。
- **言語**：コード、コメント、CLI のメッセージとヘルプは英語。README と issue / PR は日本語。
- **エラーメッセージ**は小文字で始め、利用者が次に何をすればよいかが分かるようにする（例：候補のネットワーク名を並べる、`--force` を案内する）。

## 動作確認

ユニットテストは Docker を使わない部分（引数の解析、名前の生成・検証など）だけ。Docker を操作する部分を変えたら、実際のデーモンで動かして確かめる。

- デーモンが動いていなければ `dockerd` をバックグラウンドで起動する（クラウド環境では root で起動できる）。
- ポートを公開していない Compose プロジェクト（例：postgres と redis）と、`docker run -d --network <net> --name web nginx:alpine` を用意する。
- `fumi run` は端末（TTY）が必要なので、Python の `pty` などで擬似端末を用意して操作する。
- 確認が終わったら、`fumi ls` が空になる（コンテナが残っていない）ことも見る。

## 進め方

- 機能は GitHub の issue で管理する。
- `main` から作業ブランチ（`feature/...`）を切り、PR を作ってマージする（マージコミット）。PR の説明には、実際に動かして確認したことを書く。
- 利用者から見える変更をしたら、README も同じ PR で更新する。
