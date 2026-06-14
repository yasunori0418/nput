// Package paths computes the on-disk profile layout the engine operates on
// (→ ADR-0005, ADR-0013, ADR-0022, ADR-0024, ADR-0025).
//
// profileDir は各 config 専用ディレクトリ（= flock キー）。profile リンクはその中の
// profile、世代は profile-N-link、build out-link は .pending、backref .root は
// <roothash> 階層に置く（→ docs/spec.md「profile のオンディスクレイアウト」）。
package paths

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yasunori0418/nput/internal/manifest"
)

// rootHashLen は roothash の hex 文字数（128bit・固定長・FS 安全・→ ADR-0013）。
// lib.mkManifest の anchorName（sha256 の先頭 32 hex）と桁を揃える。
const rootHashLen = 32

// StateDir は profile 群の基底 <state> を返す。$XDG_STATE_HOME があればそれ、
// 無ければ $HOME/.local/state（nix 本体の profile 既定と整合・→ ADR-0022）。
func StateDir() (string, error) {
	if s := os.Getenv("XDG_STATE_HOME"); s != "" {
		return s, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("nput: $HOME を解決できません: %w", err)
	}
	return filepath.Join(home, ".local", "state"), nil
}

// RootHash は解決後の絶対 root パスの sha256 を短縮した hex を返す（→ ADR-0013）。
func RootHash(absRoot string) string {
	sum := sha256.Sum256([]byte(absRoot))
	return hex.EncodeToString(sum[:])[:rootHashLen]
}

// Profile は 1 config 分の profile レイアウトのパス一式。
type Profile struct {
	// Dir は profileDir（config 専用ディレクトリ・flock キー）。
	Dir string
	// Profile は profile リンク（nix-env --profile の対象）。
	Profile string
	// Pending は nix build --out-link の出力先（profile を貫通しない兄弟）。
	Pending string
	// BackrefDir は backref .root を置く階層。home（--root なし）では空。
	BackrefDir string
	// Backref は元 root の絶対パスを記録する backref ファイル。home では空。
	Backref string
}

// Resolve は state 基底・config 名・rootKind・解決後絶対 root・--root 上書き有無から
// profile レイアウトを確定する（→ docs/spec.md「root の解決」表・ADR-0024, ADR-0025）。
//
//   - home（--root なし）              : <state>/nix/profiles/nput/<name>（backref なし）
//   - project / fixed / --root 上書き : <state>/nix/profiles/nput/<roothash>/<name>
//     （backref .root は <roothash> 階層）
func Resolve(stateDir, name, rootKind, absRoot string, rootOverride bool) Profile {
	base := filepath.Join(stateDir, "nix", "profiles", "nput")

	// home（--root なし）のみ <name> 直キー。それ以外は root ごとに独立系列へ分離する。
	if rootKind == manifest.RootKindHome && !rootOverride {
		dir := filepath.Join(base, name)
		return Profile{
			Dir:     dir,
			Profile: filepath.Join(dir, "profile"),
			Pending: filepath.Join(dir, ".pending"),
		}
	}

	hashDir := filepath.Join(base, RootHash(absRoot))
	dir := filepath.Join(hashDir, name)
	return Profile{
		Dir:        dir,
		Profile:    filepath.Join(dir, "profile"),
		Pending:    filepath.Join(dir, ".pending"),
		BackrefDir: hashDir,
		Backref:    filepath.Join(hashDir, ".root"),
	}
}
