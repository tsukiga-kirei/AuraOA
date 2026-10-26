package service
import("encoding/json";"os";"testing")
func TestExportRecheckContractForDocs(t *testing.T){ b,e:=json.Marshal(map[string]string{"audit":aiRecheckOutputInstructions("audit"),"archive":aiRecheckOutputInstructions("archive")});if e!=nil{t.Fatal(e)};if e=os.WriteFile("/tmp/auraoa-final-recheck-contracts.json",b,0600);e!=nil{t.Fatal(e)} }
