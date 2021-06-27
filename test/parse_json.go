package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	// derechov1alpha1 "github.com/Panlichen/cascade_operator/api/v1alpha1"
)

type SimpleType struct {
	TypeAlias string `json:"type_alias"`
}

type SimpleSpec struct {
	TypesSpec []SimpleType `json:"typesSpec"`
}

func main() {
	file, _ := os.Open("simple_test.json")
	defer file.Close()

	wr := bytes.Buffer{}
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		wr.WriteString(sc.Text())
	}

	jsonStr := wr.String()
	jsonStr = "\"typesSpec\": " + jsonStr
	jsonStr = strings.Replace(jsonStr, " ", "", -1)
	// jsonStr = strings.Replace(jsonStr, "\"", "", -1)
	fmt.Println(jsonStr)

	simpleSpec := new(SimpleSpec)
	simpleSpec.TypesSpec = make([]SimpleType, 0)
	json.Unmarshal([]byte(jsonStr), simpleSpec)
	fmt.Println(fmt.Sprintf("Unmarshal done, parse %v types", len(simpleSpec.TypesSpec)))

}

// func main() {

// 	fContent, _ := ioutil.ReadFile("test.json")
// 	jsonStr := string(fContent)
// 	jsonStr = "\"typesSpec\": " + jsonStr
// 	fmt.Println(jsonStr)

// 	cascadeNodeManagerSpec := new(derechov1alpha1.CascadeNodeManagerSpec)
// 	json.Unmarshal([]byte(jsonStr), cascadeNodeManagerSpec)
// 	fmt.Println(fmt.Sprintf("Unmarshal done, parse %v types", len(cascadeNodeManagerSpec.TypesSpec)))
// }
