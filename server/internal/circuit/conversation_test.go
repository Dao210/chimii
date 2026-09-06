package circuit
import "testing"
func TestCircuitConversationOnlySelectsReviewedProjects(t *testing.T){
 c:=StarterCatalog()
 for _,raw:=range []string{`{"outcome":"ready","project_id":"fm-radio","title":"收音机"}`,`{"outcome":"reply","project_id":"fm-radio"}`,`{"outcome":"clarify","question":"灯还是收音机？"}`,`{"outcome":"unsupported","message":"这个套装不支持录音。"}`} {if _,err:=ParseConversationDecision(raw,c);err!=nil{t.Fatal(err)}}
 for _,raw:=range []string{`{"outcome":"ready","project_id":"bluetooth-radio","title":"radio"}`,`{"outcome":"ready","project_id":"fm-radio","title":"radio","wires":[]}`,`{"outcome":"reply","message":"invented wiring"}`,`{"outcome":"clarify","question":"which?","project_id":"fm-radio"}`,`{"outcome":"reply","project_id":"fm-radio"} []`}{if _,err:=ParseConversationDecision(raw,c);err==nil{t.Fatalf("accepted %s",raw)}}
}
