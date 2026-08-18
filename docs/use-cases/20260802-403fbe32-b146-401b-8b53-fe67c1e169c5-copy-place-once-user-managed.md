---
id: "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
type: use_case
name: "リポジトリの内容を copy で初回だけ配置し、その後はユーザーが手で編集して育てる"
refines:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
---
# UC-403fbe32-b146-401b-8b53-fe67c1e169c5: リポジトリの内容を copy で初回だけ配置し、その後はユーザーが手で編集して育てる

## 使われ方

配置したあとユーザー自身が編集したいファイルを、`method = "copy"` で配置する。store への
symlink は read-only で編集できないため、この用途には copy を使う。

具体例は「カラーテーマリポジトリから特定テーマだけを `~/.local/share/themes` に配置して
手元で色を調整する」。

copy は **place-once**（初回マテリアライズしたら以後 nput は触らない）。target が既に在れば
上書きせず、編集済みの内容は不可触である。`src` 側の更新を反映したいときは
`nput apply --recopy`（全 copy target を src から無条件上書き）か、`nput reset` で撤去して
再 apply する（→ ADR-0020）。

```bash
nput apply <name> --recopy   # copy target を src から無条件に上書き再コピー
nput reset <name> [target]   # 配置物を撤去（copy も削除・確認あり）
```

copy は世代管理の対象外であり、ロールバックされない。entry が消えても copy target は自動
削除しない（手編集したデータのクロバーを避けるため）が、orphan 化は警告で通知される。

## この使われ方が要求すること

- `method` で copy を選べ、copy が世代管理の対象外になること
- place-once であり、target が既に在れば上書きしないこと
- copy されたファイルがユーザーに編集可能であること（store の read-only mode を引き継がない）
- src 側の更新を明示操作で反映できること
- 配置物を明示的に撤去する手段があること。copy は世代の外に出るため、世代を戻しても消えず
  撤去には専用の手段が要る。その手段（`reset`）は copy を消す唯一の明示手段だが、reset 自体は
  copy 専用ではなく symlink を含む配置物を home / project の両モードで撤去する
  （→ UC-f2436d68-91ff-4c48-b1df-47acefe4f464 / UC-19a90989-0ae3-438f-8a75-4e1e2637f81c）
- nput が置いていない実ファイルが target に在るとき、黙って skip せず可視化されること
- src ツリー内の symlink が deref されず構造が保たれること

## 出典

`docs/concept.md`「copy はユーザー管理の副作用」、および「世代管理（standalone）」の配置種別
表の copy 行。「想定ユースケース」home mode 節のカラーテーマの例も同じ使われ方。
