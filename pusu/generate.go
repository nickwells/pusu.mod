//go:build generate

package pusu

//go:generate stringer -type MsgType

//go:generate protoc --go_out=. --go_opt=paths=source_relative pusu.proto
