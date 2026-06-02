package spectest

type moduleTestMode int

const (
	moduleTestNone moduleTestMode = iota
	moduleTestUnlinkable
	moduleTestUninstantiable
	moduleTestInstantiate
	moduleTestRuntime
)

type moduleExpectation struct {
	mode moduleTestMode
	text string
}

func classifyModule(spec *specTest, name string) moduleExpectation {
	exp := moduleExpectation{mode: moduleTestNone}

	var current string
	for _, cmd := range spec.Commands {
		switch cmd.Type {
		case "module":
			current = cmd.Filename
			if name == current {
				exp.mode = moduleTestInstantiate
			}
		case "action", "assert_return", "assert_trap", "assert_exhaustion":
			if name == current {
				exp.mode = moduleTestRuntime
				return exp
			}
		case "assert_unlinkable":
			if cmd.Filename == name {
				exp.mode = moduleTestUnlinkable
				exp.text = cmd.Text
				return exp
			}
		case "assert_uninstantiable":
			if cmd.Filename == name {
				exp.mode = moduleTestUninstantiable
				exp.text = cmd.Text
				return exp
			}
		}
	}

	return exp
}
