# Goniko

Ninonama(ニコ生)生放送のタイムシフト配信動画をダウンロードするためのツールです。
動画 (MP4) または音声 (MP3) の保存をサポートし、Cookie管理機能も提供します。

## 特徴

-   `yt-dlp` を使用して音声か動画をダウンロード
-   `ffmpeg` を使用して音声と動画を結合
-   `cookie` サブコマンドで Cookie を保存可能

## 必要条件

-   [Go](https://go.dev/) 1.21+
-   [ffmpeg](https://ffmpeg.org/) がインストールされていること
-   [yt-dlp](https://github.com/yt-dlp/yt-dlp)がインストールされていること
-   ニコニコのCookie

## インストール方法

``` bash
git clone https://github.com/aomori446/goniko.git
cd goniko
go build -o goniko
```

## 使用方法

### 1. Cookie の設定

ブラウザから取得した **ニコニコのログイン Cookie** を保存：

``` bash
./goniko cookie --raw "your_raw_cookie_string" -o cookies.txt
```

これで Cookie が `cookies.txt` に保存されます。

### 2. 動画をダウンロード

音声付き MP4 を保存：

``` bash
./goniko -u "https://live.nicovideo.jp/watch/xxxxxx" -i cookies.txt -o output.mp4
```

### 3. 音声のみをダウンロード

MP3 形式で保存：

``` bash
./goniko -u "https://live.nicovideo.jp/watch/xxxxxx" -i cookies.txt -o output.mp3 -a
```

## コマンドライン引数

### メインコマンド

  引数   説明
  ------ ---------------------------------------------------
  `-u`   ニコ生番組の視聴 URL
  `-i`   Cookie ファイルのパス (デフォルト: `cookies.txt`)
  `-o`   出力ファイルパス (`.mp4` または `.mp3`)
  `-a`   音声のみ (MP3) を保存

### サブコマンド `cookie`

  引数      説明
  --------- -----------------------------------------------------
  `--raw`   Cookie 文字列
  `-o`      Cookie ファイルの出力先 (デフォルト: `cookies.txt`)

## 使用例

``` bash
# Cookie を保存
./goniko cookie --raw "nico_session=xxxxx;" -o cookies.txt

# 動画をダウンロード
./goniko -u "https://live.nicovideo.jp/watch/lv123456789" -o show.mp4

# 音声のみをダウンロード
./goniko -u "https://live.nicovideo.jp/watch/lv123456789" -o show.mp3 -a
```

## 注意事項

-   ニコ生のストリームはログイン必須のため、有効な Cookie が必要です。
-   現在は `.mp4` と `.mp3` のみ対応しています。
-   ffmpeg が正しくインストールされている必要があります。

------------------------------------------------------------------------

## ライセンス

MIT License
