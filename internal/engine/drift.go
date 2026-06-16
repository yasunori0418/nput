package engine

import (
	"path/filepath"

	"github.com/yasunori0418/nput/internal/planner"
)

// 世代スキップ + lstat ドリフト修復（project mode 限定・→ ADR-0005, ADR-0017,
// docs/spec.md「project mode の世代」）。
//
// devShell / direnv の shellHook はシェル再入のたびに走るため、新 link-farm
// derivation が前世代と同一なら新世代は積まない（--set を省く・世代無限増殖の回避）。
// ただし「derivation 同一 ⇒ FS 同一」は foreign tool の書き換えで崩れるため、完全
// no-op にはせず各 entry の target を lstat 検査し、ドリフトした entry だけ再張りする。

// generationUnchanged は新 link-farm が前世代（profile リンクが指す世代）と同一の store
// derivation かを返す。profile リンク → 世代リンク → link-farm store パスの連鎖を解決して
// 新 link-farm の store パスと突き合わせる。どちらかが解決できなければ error を返し、呼び出し側は
// 安全側（通常 apply = 新世代コミット）に倒す。
func generationUnchanged(profileLink, newLinkFarm string) (bool, error) {
	prev, err := filepath.EvalSymlinks(profileLink)
	if err != nil {
		return false, err
	}
	next, err := filepath.EvalSymlinks(newLinkFarm)
	if err != nil {
		return false, err
	}
	return prev == next, nil
}

// repairDrift は世代スキップ時の lstat ドリフト修復を行う。planner が現 FS 状態から算出した
// プランのうち、ドリフトした entry **だけ** を再収束させ、stale 除去も世代コミットもしない。
//
//   - symlink: PlaceNew（target 消失）と PlaceForeign（foreign 書き換え）だけ再張りする。
//     PlaceReplace は「記録通りの先を指す symlink」= ドリフトなしのため触らない（→ ADR-0017）。
//     foreign 書き換えの warning は呼び出し側が emitWarnings(plan.Warnings) で既に出している。
//   - copy: planner.Copies は target 不在の place-once アクションのみを含むため、それを反映すると
//     「copy は target 不在時のみ place-once 復帰」になる。内容差（ユーザー編集）は CopyAction が
//     生まれず触らない（→ docs/spec.md「project mode の世代」・ADR-0022）。--recopy のときは
//     recopyAll で無条件上書きする（recopy は世代外の opt-in）。
func (a *applier) repairDrift(plan planner.Plan, recopy bool) error {
	drifted := make([]planner.PlaceAction, 0, len(plan.Place))
	for _, act := range plan.Place {
		if act.Kind == planner.PlaceReplace {
			continue // 記録通りの先を指す symlink = in-sync。ドリフトなしで再張り不要。
		}
		drifted = append(drifted, act)
	}
	if err := a.place(drifted); err != nil {
		return err
	}
	return a.materializeCopies(plan, recopy)
}
