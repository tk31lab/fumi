# fumi

`fumi`（踏み）は、Docker ネットワーク上に使い捨ての踏み台コンテナを立てて、そのシェルに入るための CLI です。

Docker Compose で起動した DB のように、ホストにポートを公開していないサービスへ、同じネットワークの中から接続できます。

```console
$ fumi networks
NAME            DRIVER   PROJECT   CONTAINERS
myapp_default   bridge   myapp     cache, db

$ fumi run -n myapp_default
Started fumi-shizuka-tsuki on network myapp_default (exit the shell to remove it)
Mounted /home/me/myapp at /work
fumi-shizuka-tsuki:/work # nc -zv db 5432
db (172.18.0.3:5432) open
```

シェルを抜けると、コンテナは自動で削除されます。

## インストール

```sh
go install github.com/tk31lab/fumi@latest
```

Go 1.27 以上が必要です。

## 使い方

### `fumi run` — 踏み台を起動する

```sh
fumi run --network <NETWORK> [--image IMAGE] [--shell SHELL] [--name NAME]
```

| フラグ | デフォルト | 説明 |
|---|---|---|
| `-n`, `--network` | （必須） | 参加させるネットワーク |
| `--image` | `alpine:3.24` | 使うイメージ。ローカルになければ pull します |
| `-s`, `--shell` | `sh` | 起動するシェル |
| `--name` | 自動生成 | コンテナ名。ホスト名にも使います |

- 実行したときのカレントディレクトリを、コンテナの `/work` に読み書きできる状態でマウントします（`-v "$PWD:/work"` 相当）。シェルも `/work` から始まるので、手元の SQL ファイルを流したり、ダンプを手元に書き出したりできます。
- コンテナ名を指定しないと、日本庭園にちなんだ名前（`fumi-<ようす>-<庭にあるもの>`、例：`fumi-oboro-kiri`）を自動で付けます。
- 起動したコンテナには、ラベル `fumi.managed=true` が付きます。
- 終了コードは、シェルの終了コードをそのまま返します。
- 対話的に使う前提なので、標準入力と標準出力が端末である必要があります。

### `fumi networks` — ネットワーク一覧（エイリアス：`net`）

```sh
fumi networks [--all] [--format table|json]
```

ネットワークごとに、Compose のプロジェクト名と、つながっている起動中のコンテナ（Compose のサービス名があればその名前）を表示します。
Docker が最初から用意している `bridge` / `host` / `none` は、`--all` を付けたときだけ表示します。

### `fumi completion` — シェル補完

`--network` を Tab で補完できます（いま存在するネットワーク名が候補になります）。

```sh
# bash（~/.bashrc に追記）
source <(fumi completion bash)

# zsh（~/.zshrc に追記）
source <(fumi completion zsh)
```

fish と PowerShell にも対応しています（`fumi completion --help`）。

## Docker への接続

`DOCKER_HOST` などの環境変数に従い、未設定なら `/var/run/docker.sock` に接続します。
`/work` にマウントされるのは Docker デーモンが動いているマシン上のパスなので、`DOCKER_HOST` でリモートのデーモンにつないでいる場合は、手元のディレクトリはマウントされません。

`docker context` の設定は読まないので、Colima や OrbStack などでソケットの場所が違う場合は `DOCKER_HOST` を設定してください。

```sh
export DOCKER_HOST=$(docker context inspect --format '{{.Endpoints.docker.Host}}')
```

## ライセンス

MIT
