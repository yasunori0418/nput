---
id: "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
type: use_case
name: "役割ごとに config を分けて更新を独立させ、1 つの更新を他の役割へ波及させない"
refines:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
---
# UC-1c280dce: 役割ごとに config を分けて更新を独立させ、1 つの更新を他の役割へ波及させない

## 使われ方

home-manager のように全体を一括管理するのではなく、用途ごとに独立した配置単位を定義する。
各配置単位は `nput.<name>` で識別され（1 つ = 1 profile）、独立した profile・世代系列として
管理される（→ ADR-0014）。

```bash
# 役割ごとに独立して更新・適用できる（それぞれ別 profile）
nput apply vim-plugins
nput apply zsh-plugins
nput apply claude-skills
```

`src` を更新（flake input の更新 / npins update 等）した後、対象 config だけを再適用する
ことで、他のツールへの影響を完全に排除できる。vim プラグインの更新が zsh の設定を壊す
「共倒れ」が構造的に起こらない。

逆に、複数 config をまとめて適用したい場面もある。`nput apply --all` は全 config を順に
適用し、途中で 1 つが失敗しても残りを続行して最後に集約する。config を分けたことが、
一括適用を諦める理由にはならない。

この使われ方は配置モードに依らない。home mode でも project mode でも、role ごとの config
分割と個別適用は同じ形で成立する（→ UC-f2436d68 / UC-19a90989）。

## この使われ方が要求すること

- config が `nput.<name>` で名前づけられ、CLI が名前で config を選べること
- config ごとに独立した profile を持ち、更新が他の config へ波及しないこと
- 全 config をまとめて適用する手段があり、部分失敗でも残りが続行されること
- 同一 target を複数 config が狙ったときの振る舞いが決まっていること
- entry 単位ではなく config 単位が独立の粒度であること（一部 entry だけの適用は提供しない）

## 出典

`docs/concept.md`「独立した更新サイクル」、および「コンセプトの核心」の「役割を分離し、
各役割を独立して管理・更新できるようにする」。
