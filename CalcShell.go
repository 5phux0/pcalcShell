package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"pcalc"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type pElement struct {
	exp  *pcalc.Expression
	inds [2]int
}

const (
	o_inv = iota - 1
	o_pow
	o_npow
	o_mul
	o_nmul
	o_div
	o_ndiv
	o_add
	o_sub
)

var mathematicalConstants = make(map[string]*pcalc.Expression)
var chemicalConstants = make(map[string]*pcalc.Expression)
var physicalConstants = make(map[string]*pcalc.Expression)
var userVars = make(map[string]*pcalc.Expression)
var standardFuncs = make(map[string]*pcalc.Expression)
var selectedNamespace = &userVars

//Parse expressions in the following order
var nonfuncParRe = regexp.MustCompile(`([\pN]*)\(`)
var stdfuncParRE = regexp.MustCompile(`^.[\pL\pN]+\(`)
var nsfuncRE = regexp.MustCompile(`([\pL]+[\pN]*){0,}\.([\pL]+[\pN]*){1,}`)
var unknownRE = regexp.MustCompile(`([\pL]+[\pN]*){1,}`)
var sdFloatRE = regexp.MustCompile(`([\pN]+\.[\pN]*)|([\pN]*\.[\pN]+)`)
var intRE = regexp.MustCompile(`[\pN]+`)

func main() {
	setupConstantExps()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if parseLine(scanner.Text()) {
			return
		}
		fmt.Println()
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
	}
}

func setupConstantExps() {
	standardFuncs["ln"] = pcalc.NaturalLogarithmOfExpression(pcalc.NewExpressionWithUnknown("a"))
	standardFuncs["lg"] = pcalc.DecimalLogarithmOfExpression(pcalc.NewExpressionWithUnknown("a"))

	chemicalConstants["R"] = pcalc.NewExpressionWithConstant(pcalc.MakeSDFloat(8.3144, 5))
	chemicalConstants["r"] = chemicalConstants["R"]
	chemicalConstants["F"] = pcalc.NewExpressionWithConstant(pcalc.MakeSDFloat(96485.33, 7))
	chemicalConstants["f"] = chemicalConstants["F"]

	physicalConstants["C"] = pcalc.NewExpressionWithConstant(pcalc.MakeSDFloat(299792458, 9))
	physicalConstants["c"] = physicalConstants["C"]
	physicalConstants["G"] = pcalc.NewExpressionWithConstant(pcalc.MakeSDFloat(9.80665, 6))
	physicalConstants["g"] = physicalConstants["G"]

	mathematicalConstants["pi"] = pcalc.NewExpressionWithConstant(pcalc.MakeSDFloat(math.Pi, 15))
	mathematicalConstants["e"] = pcalc.NewExpressionWithConstant(pcalc.MakeSDFloat(math.E, 15))
}

func parseLine(line string) bool {
	s := strings.TrimSpace(line)
	if len(s) >= 4 && s[:4] == "exit" {
		return true
	}

	if ind := strings.Index(s, "list "); ind != -1 {
		vn := fmt.Sprintf("%c%v", s[ind+5], ".")
		ns, _ := parseVarName(vn)
		if ns != nil {
			for k, v := range *ns {
				fmt.Printf("%v=%v\n", k, v.Description())
			}
		}
	} else if ind := strings.Index(s, "<<"); ind != -1 {
		ns, key := parseVarName(s[:ind])
		exp := parseExpression(s[ind+2:])
		if exp != nil {
			if ns != &userVars {
				fmt.Printf("Can't assign to constant namespace. Assigning to u.%s instead.\n", key)
			}
			userVars[key] = exp
		} else {
			fmt.Println("Didn't assign. Expression = nil")
		}
	} else {
		exp := parseExpression(s)
		if exp == nil {
			fmt.Println("Entered expression = nil")
			return false
		}
		if v := exp.Value(); len(exp.ListUnknowns()) > 0 || v == nil {
			fmt.Println(exp.Description())
		} else {
			fmt.Println(exp.Description(), "=", exp.Value().Description())
		}
	}
	return false
}

func parseVarName(s string) (nameSpace *map[string]*pcalc.Expression, key string) {
	s = strings.TrimSpace(s)
	nameSpace = nil
	key = ""
	fields := strings.Split(s, ".")
	if len(fields) == 1 {
		nameSpace = &standardFuncs
		key = fields[0]
	} else if len(fields) == 2 {
		switch fields[0] {
		case "":
			nameSpace = selectedNamespace
		case "u":
			nameSpace = &userVars
		case "m":
			nameSpace = &mathematicalConstants
		case "c":
			nameSpace = &chemicalConstants
		case "p":
			nameSpace = &physicalConstants
		default:
			fmt.Printf("Invalid namespace:%s\n", fields[0])
		}
		key = fields[1]
	} else {
		fmt.Printf("A reference to an expression can't contain more than one \".\"")
	}
	return
}

func parseArgumentString(s string, unknowns []string) *map[string]*pcalc.Expression {
	ret := make(map[string]*pcalc.Expression, 0)
	fields := strings.Split(s, ",")

	for _, v := range fields {
		subF := strings.Split(v, "=")
		if len(subF) == 2 {
			k := strings.TrimSpace(subF[0])
			for _, u := range unknowns {
				if u == k {
					ret[k] = parseExpression(strings.TrimSpace(subF[1]))
					break
				}
			}
		}
		if len(subF) > 2 {
			fmt.Println("One \"=\" per argument assignment.")
			return &ret
		}
	}
	for _, v := range fields {
		if len(ret) >= len(unknowns) {
			return &ret
		}
		if strings.Index(v, "=") == -1 {
			for _, u := range unknowns {
				if _, e := ret[u]; !e {
					ret[u] = parseExpression(strings.TrimSpace(v))
					break
				}
			}
		}

	}
	return &ret
}

func parseExpression(s string) *pcalc.Expression {
	values := parseParenthesisEnclosedExpressions(s, 0)
	exps := make([]*pcalc.Expression, len(values))
	opStrings := make([]string, len(values))
	operators := make([]int, len(values))
	for i, v := range values {
		exps[i] = v.exp
	}

	lasti := 0
	var i int
	for i = range values {
		opStrings[i] = s[lasti:values[i].inds[0]]
		lasti = values[i].inds[1]
	}
	for i, v := range opStrings {
		foundOp := -1
		if len(v) == 0 && i != 0 {
			foundOp = o_mul
		}
		fields := strings.Split(v, "")
		for _, c := range fields {
			if c == " " {
				continue
			}
			if foundOp == -1 {
				switch c {
				case "^":
					foundOp = o_pow
				case "*":
					foundOp = o_mul
				case "/":
					foundOp = o_div
				case "+":
					foundOp = o_add
				case "-":
					foundOp = o_sub
				default:
					fmt.Printf("Invalid operator: \"%s\"\n", v)
					return nil
				}
			} else {
				if c == "-" {
					if foundOp%2 == 0 {
						foundOp++
					} else {
						foundOp--
					}
				} else {
					fmt.Printf("Invalid operator: \"%s\"\n", v)
					return nil
				}
			}
		}
		operators[i] = foundOp
	}

	//^ loop
	for i := 1; i < len(operators); {
		if v := operators[i]; !(v == o_pow || v == o_npow) {
			i++
			continue
		}
		var ne *pcalc.Expression
		if v := operators[i]; v == o_pow {
			ne = pcalc.RaiseExpressionToPower(exps[i-1], exps[i])
		} else if v == o_npow {
			ne = pcalc.RaiseExpressionToPower(exps[i-1], pcalc.SignInvertedExpression(exps[i]))
		}
		exps = append(exps[:i], exps[i+1:]...)
		exps[i-1] = ne
		operators = append(operators[:i], operators[i+1:]...)
	}
	//*/ loop
	for i := 1; i < len(operators); {
		if v := operators[i]; v < o_mul || v > o_ndiv {
			i++
			continue
		}
		var ne *pcalc.Expression
		if v := operators[i]; v == o_mul {
			ne = pcalc.MultiplyExpressions(exps[i-1], exps[i])
		} else if v == o_nmul {
			ne = pcalc.MultiplyExpressions(exps[i-1], pcalc.SignInvertedExpression(exps[i]))
		} else if v == o_div {
			ne = pcalc.DivideExpressions(exps[i-1], exps[i])
		} else if v == o_ndiv {
			ne = pcalc.DivideExpressions(exps[i-1], pcalc.SignInvertedExpression(exps[i]))
		}
		exps = append(exps[:i], exps[i+1:]...)
		exps[i-1] = ne
		operators = append(operators[:i], operators[i+1:]...)
	}
	//+- loop
	if len(operators) > 0 {
		if operators[0] == o_sub {
			exps[0] = pcalc.SignInvertedExpression(exps[0])
		} else if operators[0] != o_inv {
			fmt.Printf("Ignoring operator \"%s\" perceeding expression\n", opStrings[0])
		}
	}
	for i := 1; i < len(operators); {
		if v := operators[i]; !(v == o_add || v == o_sub) {
			i++
			continue
		}
		var ne *pcalc.Expression
		if v := operators[i]; v == o_add {
			ne = pcalc.AddExpressions(exps[i-1], exps[i])
		} else if v == o_sub {
			ne = pcalc.SubtractExpressions(exps[i-1], exps[i])
		}
		exps = append(exps[:i], exps[i+1:]...)
		exps[i-1] = ne
		operators = append(operators[:i], operators[i+1:]...)
	}
	if len(exps) == 0 {
		return nil
	} else {
		return exps[0]
	}
}

func parseParenthesisEnclosedExpressions(s string, starti int) []pElement {
	nextParseFunction := parseStandardExpressions
	ret := make([]pElement, 0)
	ind := nonfuncParRe.FindStringIndex(s)
	if len(ind) == 0 {
		ret = nextParseFunction(s, starti)
	} else if ind[0] != 0 && unicode.IsLetter(rune(s[ind[0]-1])) {
		ret = nextParseFunction(s, starti)
	} else {
		si := ind[0]
		ind = firstParenthesisIndex(s[ind[0]:])
		ind[0] = ind[0] + si
		ind[1] = ind[1] + si
		if ind[0] > 0 {
			ret = append(ret, nextParseFunction(s[:ind[0]], starti)...)
		}
		if ind[0] < ind[1] {
			var exp *pcalc.Expression
			if ind[0]+1 >= ind[1]-1 {
				exp = nil
				fmt.Println("Type something in your parenthesis enclosed expressions, you dudwanker!")
			} else {
				exp = parseExpression(s[ind[0]+1 : ind[1]-1])
			}
			ne := pElement{pcalc.ParenthesisEnclosedExpression(exp), [2]int{starti + ind[0], starti + ind[1]}}
			ret = append(ret, ne)
		}
		if ind[1] < len(s) {
			ret = append(ret, parseParenthesisEnclosedExpressions(s[ind[1]:], starti+ind[1])...)
		}
	}
	return ret
}

func parseStandardExpressions(s string, starti int) []pElement {
	nextParseFunction := parseNonStandardExpression
	ret := make([]pElement, 0)
	ind := stdfuncParRE.FindStringIndex(s)
	if len(ind) == 0 {
		ret = nextParseFunction(s, starti)
	} else {
		ind[1] = ind[1] - 1
		pInd := firstParenthesisIndex(s[ind[1]:])
		pInd[0] = pInd[0] + ind[1]
		pInd[1] = pInd[1] + ind[1]
		if ind[0] > 0 {
			ret = append(ret, nextParseFunction(s[:ind[0]], starti)...)
		}
		if ind[0] < ind[1] {
			_, key := parseVarName(s[ind[0]:ind[1]])
			exp := standardFuncs[key]
			if exp != nil {
				var argmap *map[string]*pcalc.Expression
				if pInd[0]+1 >= pInd[1]-1 {
					a := make(map[string]*pcalc.Expression)
					argmap = &a
				} else {
					argmap = parseArgumentString(s[pInd[0]+1:pInd[1]-1], exp.ListUnknowns())
				}
				exp = exp.Call(argmap)
			} else {
				fmt.Println("Unknown expression:", s[ind[0]:ind[1]])
			}
			ne := pElement{exp, [2]int{starti + ind[0], starti + pInd[1]}}
			ret = append(ret, ne)
		}
		if pInd[1] < len(s) {
			ret = append(ret, parseStandardExpressions(s[ind[1]:], starti+ind[1])...)
		}
	}
	return ret
}

func parseNonStandardExpression(s string, starti int) []pElement {
	nextParseFunction := parseUnknownExpressions
	ret := make([]pElement, 0)
	ind := nsfuncRE.FindStringIndex(s)
	if len(ind) == 0 {
		ret = nextParseFunction(s, starti)
	} else {
		var pInd = []int{ind[0], ind[1]}
		if ind[0] > 0 {
			ret = append(ret, nextParseFunction(s[:ind[0]], starti)...)
		}
		if ind[0] < ind[1] {
			ns, key := parseVarName(s[ind[0]:ind[1]])
			var exp *pcalc.Expression
			if ns != nil {
				exp = (*ns)[key]
				if exp != nil {
					var argmap *map[string]*pcalc.Expression
					if ind[1] < len(s) && s[ind[1]] == '(' {
						pInd = firstParenthesisIndex(s[ind[1]:])
						pInd[0] = pInd[0] + ind[1]
						pInd[1] = pInd[1] + ind[1]
						if pInd[0]+1 >= pInd[1]-1 {
							a := make(map[string]*pcalc.Expression)
							argmap = &a
						} else {
							argmap = parseArgumentString(s[pInd[0]+1:pInd[1]-1], exp.ListUnknowns())
						}
						exp = exp.Call(argmap)
					}
				} else {
					fmt.Println("Unknown expression:", s[ind[0]:pInd[1]])
				}
			}
			ne := pElement{exp, [2]int{starti + ind[0], starti + pInd[1]}}
			ret = append(ret, ne)
		}
		if pInd[1] < len(s) {
			ret = append(ret, parseNonStandardExpression(s[ind[1]:], starti+ind[1])...)
		}
	}
	return ret
}

func parseUnknownExpressions(s string, starti int) []pElement {
	nextParseFunction := parseFloatExpressions
	ret := make([]pElement, 0)
	ind := unknownRE.FindStringIndex(s)
	if len(ind) == 0 {
		ret = nextParseFunction(s, starti)
	} else {
		if ind[0] > 0 {
			ret = append(ret, nextParseFunction(s[:ind[0]], starti)...)
		}
		if ind[0] < ind[1] {
			var exp *pcalc.Expression
			exp = pcalc.NewExpressionWithUnknown(s[ind[0]:ind[1]])
			ne := pElement{exp, [2]int{starti + ind[0], starti + ind[1]}}
			ret = append(ret, ne)
		}
		if ind[1] < len(s) {
			ret = append(ret, parseUnknownExpressions(s[ind[1]:], starti+ind[1])...)
		}
	}
	return ret
}

func parseFloatExpressions(s string, starti int) []pElement {
	nextParseFunction := parseIntExpressions
	ret := make([]pElement, 0)
	ind := sdFloatRE.FindStringIndex(s)
	if len(ind) == 0 {
		ret = nextParseFunction(s, starti)
	} else {
		if ind[0] > 0 {
			ret = append(ret, nextParseFunction(s[:ind[0]], starti)...)
		}
		if ind[0] < ind[1] {
			var exp *pcalc.Expression
			in, err := strconv.ParseFloat(s[ind[0]:ind[1]], 64)
			if err != nil {
				fmt.Println(err)
				exp = nil
			} else {
				num := pcalc.MakeSDFloat(in, uint8(ind[1]-ind[0]-1))
				exp = pcalc.NewExpressionWithConstant(num)
			}
			ne := pElement{exp, [2]int{starti + ind[0], starti + ind[1]}}
			ret = append(ret, ne)
		}
		if ind[1] < len(s) {
			ret = append(ret, parseFloatExpressions(s[ind[1]:], starti+ind[1])...)
		}
	}
	return ret
}

func parseIntExpressions(s string, starti int) []pElement {
	ret := make([]pElement, 0)
	ind := intRE.FindStringIndex(s)
	if len(ind) == 0 {
		return ret
	} else {
		if ind[0] < ind[1] {
			var exp *pcalc.Expression
			in, err := strconv.ParseInt(s[ind[0]:ind[1]], 10, 64)
			if err != nil {
				fmt.Println(err)
				exp = nil
			} else {
				num := pcalc.MakeFraction(in, 1)
				exp = pcalc.NewExpressionWithConstant(num)
			}
			ne := pElement{exp, [2]int{starti + ind[0], starti + ind[1]}}
			ret = append(ret, ne)
		}
		if ind[1] < len(s) {
			ret = append(ret, parseIntExpressions(s[ind[1]:], starti+ind[1])...)
		}
	}
	return ret
}

//Utillity
func firstParenthesisIndex(s string) []int {
	ret := []int{-1, -1}
	pc := 0
	for i, v := range s {
		switch v {
		case '(':
			pc++
			if ret[0] == -1 {
				ret[0] = i
			}
		case ')':
			if pc > 0 {
				pc--
				if pc == 0 {
					ret[1] = i + 1
					return ret
				}
			}
		}
	}
	ret[1] = len(s)
	return ret
}
