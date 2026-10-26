package service

import (
 "encoding/json"
 "net/http/httptest"
 "os"
 "strings"
 "testing"

 "auraoa/go-service/internal/model"
 "auraoa/go-service/internal/pkg/ai"
 "github.com/gin-gonic/gin"
 "github.com/google/uuid"
 "gorm.io/datatypes"
)

func TestAIRecheckStrictOutputContract(t *testing.T) {
 valid:=`{"reuse_original":false,"reason":"修正分数与结论","final_result":`+finalAuditRecheck+`}`
 for name,raw:=range map[string]string{
  "沿用初审":`{"reuse_original":true}`,
  "完整修改":valid,
 } {t.Run(name,func(t *testing.T){if err:=validateAIRecheckEnvelope([]byte(raw),"audit");err!=nil{t.Fatal(err)}})}
 for name,raw:=range map[string]string{
  "缺少采用标记":`{"reason":"说明","final_result":`+finalAuditRecheck+`}`,
  "沿用时多带结果":`{"reuse_original":true,"final_result":`+finalAuditRecheck+`}`,
  "缺少修改结果":`{"reuse_original":false,"reason":"说明"}`,
  "缺少说明":`{"reuse_original":false,"final_result":`+finalAuditRecheck+`}`,
  "说明全空白":strings.Replace(valid,"修正分数与结论","   ",1),
  "重复键":strings.Replace(valid,`"reuse_original":false`,`"reuse_original":true,"reuse_original":false`,1),
  "转义重复键":strings.Replace(valid,`"reason":`,`"\u0072eason":"重复","reason":`,1),
  "前缀说明":"结果如下："+valid,
  "代码围栏":"```json\n"+valid+"\n```",
  "多个对象":valid+`{}`,
  "未知顶层字段":strings.Replace(valid,`"reuse_original":false`,`"reuse_original":false,"extra":1`,1),
  "未知结果字段":strings.Replace(valid,`"overall_score":75`,`"overall_score":75,"score":75`,1),
  "分数为字符串":strings.Replace(valid,`"overall_score":75`,`"overall_score":"75"`,1),
  "分数为小数":strings.Replace(valid,`"overall_score":75`,`"overall_score":75.0`,1),
  "分数越界":strings.Replace(valid,`"overall_score":75`,`"overall_score":101`,1),
  "布尔缺失":strings.Replace(valid,`"passed":false,`,"",1),
  "布尔为字符串":strings.Replace(valid,`"passed":false`,`"passed":"false"`,1),
  "风险数组为null":strings.Replace(valid,`["金额冲突未排除"]`,`null`,1),
  "风险项为null":strings.Replace(valid,`["金额冲突未排除"]`,`[null]`,1),
  "扣分缺少points":strings.Replace(valid,`"confidence":90`,`"confidence":90,"deductions":[{"item":"问题","reason":"说明"}]`,1),
 } {t.Run(name,func(t *testing.T){if err:=validateAIRecheckEnvelope([]byte(raw),"audit");err==nil{t.Fatal("非法格式不能进入业务解析")}})}
}

func TestAIRecheckReuseAndSemanticComparison(t *testing.T) {
 c,_:=gin.CreateTestContext(httptest.NewRecorder());c.Request=httptest.NewRequest("POST","/",nil)
 caller:=&fakeRecheckCaller{response:`{"reuse_original":true}`}
 saves:=0
 result,record,err:=runAIRecheck(c,caller,uuid.New(),uuid.New(),nil,nil,&model.AIRecheckConfig{Enabled:true,UserPrompt:"{{rules}}\n{{initial_result}}"},&ai.ChatRequest{RequestType:"audit"},"",json.RawMessage(initialAuditRecheck),false,func(*model.AIRecheckRecord)error{saves++;return nil},AIRecheckInput{Rules:"本次绑定的规则"})
 if err!=nil || string(result)!=initialAuditRecheck || !record.ReuseOriginal || record.Changed || len(record.FinalResult)>0 || saves!=2 {t.Fatalf("沿用初审必须成功结束并避免重存最终副本：%+v %v",record,err)}
 if !strings.Contains(caller.req.UserPrompt,"本次绑定的规则") {t.Fatal("规则变量没有使用当次版本内容")}
 if !reviewResultsEqual(json.RawMessage(`{"risk_points":["甲","乙"],"reason":"已有  说明"}`),json.RawMessage(`{"reason":"已有 说明","risk_points":["乙","甲"]}`)) {t.Fatal("条目重排或纯空白不能误报业务调整")}
 if reviewResultsEqual(json.RawMessage(initialAuditRecheck),json.RawMessage(finalAuditRecheck)){t.Fatal("实际业务变化不能被忽略")}
}

func TestAIRecheckVisibleContractMatchesActualRequest(t *testing.T) {
 review,err:=os.ReadFile("../../../docs/reviews/2026-10-26-audit-archive-prompts-for-review.md");if err!=nil{t.Fatal(err)}
 for _,module:=range []string{"audit","archive"} {
  template:=withAIRecheckOutputInstructions(model.SystemPromptTemplate{PromptKey:module+"_system_recheck",PromptType:"system",Phase:"recheck",Content:"用户可编辑正文"},module)
  req:=buildAIRecheckRequest(&model.AIRecheckConfig{SystemPrompt:template.Content,UserPrompt:"{{initial_result}}"},&ai.ChatRequest{RequestType:module},"",json.RawMessage(initialAuditRecheck))
  if req.SystemPrompt!=template.Content+"\n\n"+template.OutputInstructions {t.Fatal("页面展示与实际追加内容不一致")}
  if !strings.Contains(string(review),template.OutputInstructions){t.Fatal("审核稿必须包含后端完整输出约定")}
  schemaRaw:=strings.SplitN(template.OutputInstructions,"【JSON Schema】\n",2)
  if len(schemaRaw)!=2 || !json.Valid([]byte(schemaRaw[1])) {t.Fatal("输出结构必须是完整合法的JSON Schema")}
 }
}

func TestRecheckScaleAndPendingPlan(t *testing.T) {
 for _,scale:=range []string{"strict","standard","loose"} {
  var data map[string]json.RawMessage
  raw,err:=normalizeSavedRecheckConfig([]byte(`{"audit_strictness":"`+scale+`","recheck":{"enabled":true}}`));if err!=nil{t.Fatal(err)}
  json.Unmarshal(raw,&data);var cfg model.AIRecheckConfig;json.Unmarshal(data["recheck"],&cfg)
  if cfg.SystemPrompt!=recheckSystemPromptForStrictness(scale) {t.Fatal("保存缺省值必须选取本次尺度")}
  v:=&model.ExecutionConfigVersion{ConfigSnapshot:datatypes.JSON(`{"ai_config":`+string(raw)+`}`)}
  if string(pendingAIRecheck(v))!=`{"enabled":true}`{t.Fatal("入队时应记录实际版本的复核计划")}
 }
 custom:=&model.AIRecheckConfig{SystemPrompt:"自定义正文"}
 if recheckConfigForStrictness(custom,"standard","strict").SystemPrompt!=custom.SystemPrompt{t.Fatal("切换尺度不能覆盖自定义复核提示词")}
 builtin:=&model.AIRecheckConfig{SystemPrompt:defaultRecheckSystemPrompt}
 if recheckConfigForStrictness(builtin,"standard","loose").SystemPrompt!=looseRecheckSystemPrompt || builtin.SystemPrompt!=defaultRecheckSystemPrompt {t.Fatal("切换默认要求必须生成副本并保留旧配置")}
}
