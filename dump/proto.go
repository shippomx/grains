package dump

import (
	"regexp"
	"strconv"
	"strings"
)

func (f *Frame) decodeHead(header string) {
	// goroutine [0-9]* \[[\w ]*, [0-9]* minutes]
	reg := regexp.MustCompile(`goroutine ([\d]+) \[([\w ._]+), ([\d]+) minutes]:`)
	//reg := regexp.MustCompile(`goroutine ([\d]) \[([\w ]), ([\d]) minutes]:`)
	params := reg.FindStringSubmatch(header)
	if len(params) == 4 {
		head := Head{}
		head.GID, _ = strconv.Atoi(params[1])
		head.Duration, _ = strconv.Atoi(params[3])
		f.Head = head
		f.Reason = params[2]
		return
	}

	// goroutine [0-9]* \[[\w ]*]
	reg = regexp.MustCompile(`goroutine ([\d]+) \[([\w ._]+)]:`)
	params = reg.FindStringSubmatch(header)
	if len(params) == 3 {
		head := Head{}
		head.GID, _ = strconv.Atoi(params[1])
		f.Head = head
		f.Reason = params[2]
		return
	}
	return
}

func  (f *Frame) decodeBody(body []string) {
	for i := 0; i < len(body) - 1; i += 2 {
		stack := Stack{}
		reg := regexp.MustCompile(`([\D\w.\-\/\(\*\)]+)\(([0x\w, ]+)\)`)
		funcAndParams := reg.FindStringSubmatch(body[i])
		if len(funcAndParams) == 3 {
			stack.FuncName = funcAndParams[1]
			stack.Params = funcAndParams[2]
		} else {
			stack.FuncName = body[i]
		}
		if len(body[i+1]) > 0 {
			strLoc := body[i+1][1:]
			stack.Location = strLoc
			reg = regexp.MustCompile(`([\w.\-\/:\d]+) ([+0x\d]+)`)
			locations := reg.FindStringSubmatch(strLoc)
			if len(locations) > 0 {
				stack.Location = locations[0]
			}
		}
		f.Stacks = append(f.Stacks, stack)
	}
	f.Size = len(body)
	f.checkHoldLock()

	return
}

func (f *Frame)checkHoldLock() {
	if len(f.Stacks) < 2 {
		return
	}
	for i := len(f.Stacks) - 2; i >= 0 ; i -- {
		funcName := f.Stacks[i].FuncName
		tfunc := strings.ToLower(funcName)
		if strings.Contains(tfunc, ".lock") {
			f.LockInfo.Stack = &f.Stacks[i+1]
			f.LockType = funcName
			reg := regexp.MustCompile(`\(([\w.*]+)\)`)
			for j := i+1; j < len(f.Stacks); j++ {
				//if !strings.Contains(f.Stacks[j].FuncName, "unlock") {
				holdFunc := f.Stacks[j].FuncName
				holders := reg.FindStringSubmatch(holdFunc)
				if len(holders) > 1 {
					tHolder := holders[1]
					//if len(f.LockHolders) == 0 {
					//	f.LockHolders = append(f.LockHolders, tHolder)
					//}
					find := false
					for _, holder := range f.LockHolders {
						if holder == tHolder {
							find = true
							break
						}
					}
					if !find {
						f.LockHolders = append(f.LockHolders, tHolder)
					}
				}
				//}
			}
			return
		}
	}
}

func (f *Frame) hasHighSimilarity(f2 *Frame) (ret bool) {
	if len(f.Stacks) != len(f2.Stacks) {
		return false
	}
	for i := 0; i < len(f.Stacks); i++ {
		if f.Stacks[i].FuncName == f2.Stacks[i].FuncName &&
			f.Stacks[i].Location == f2.Stacks[i].Location {
				continue
		} else {
			return false
		}
	}
	return true
}
