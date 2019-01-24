package main

var (
	BuildTimestamp = "DEBUG (no info)"
	GitHash        = "DEBUG (no git hash)"
)

func PrintLogo() {
	println(`

  ╭───────╮ ──────── ╭───────╮  
  │                  │       │ 
  │   ∙━━━┯━━━━━━━━━━┿━━━∙   │
  │       │          │       │ 
  ╰───────╯ ──────── ╰───────╯

            PROTOCOL
            OBSERVER
	`)
}

func PrintVersionDigest() {
	println(
		"\n",
		"Buildstamp:", BuildTimestamp, "\n",
		"  Git hash:", GitHash, "\n",
		"\n\n")
}
