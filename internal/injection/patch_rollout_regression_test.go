package injection

import (
 "reflect"
 "testing"

 resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// These regression tests intentionally fail on the current implementation.
// A changing pod template with unchanged inputs would trigger a new ReplicaSet
// if persisted to a Deployment. No Kubernetes cluster is required.
func rolloutProvider(withEnv, withConfig bool) resourcesv1alpha1.ResourceProvider {
 p := resourcesv1alpha1.ResourceProvider{}
 p.Name = "nats"
 p.Spec.Type = "nats"
 if withEnv {
  p.Spec.Outputs.Env = map[string]string{
   "NATS_URL":"nats://nats:4222", "NATS_HOST":"nats", "NATS_PORT":"4222",
   "NATS_TIMEOUT":"10s", "NATS_QUEUE":"queue", "NATS_TOPIC":"topic",
   "NATS_REPLY":"reply", "NATS_ENABLED":"true",
  }
 }
 if withConfig {
  p.Spec.Outputs.Config = []resourcesv1alpha1.ConfigMapOutput{{
   NameTemplate:"app-config", Env:map[string]string{
    "CONFIG_A":"a", "CONFIG_B":"b", "CONFIG_C":"c", "CONFIG_D":"d",
    "CONFIG_E":"e", "CONFIG_F":"f", "CONFIG_G":"g", "CONFIG_H":"h",
   },
  }}
 }
 return p
}

func podTemplate(t *testing.T, obj *unstructured.Unstructured) map[string]any {
 t.Helper()
 template, found, err := unstructured.NestedMap(obj.Object, "spec", "template")
 if err != nil || !found { t.Fatalf("pod template missing: %v", err) }
 return template
}

func envOrder(t *testing.T, obj *unstructured.Unstructured) []string {
 t.Helper()
 env, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
 if err != nil || !found { t.Fatalf("containers missing: %v", err) }
 c := env[0].(map[string]any)
 out := []string{}
 for _, item := range c["env"].([]any) { out = append(out, item.(map[string]any)["name"].(string)) }
 return out
}

func TestRolloutRegression_EnvMapRepeatedReconcileDoesNotChangeTemplate(t *testing.T) {
 assertStableTemplateAcrossReconciles(t, rolloutProvider(true, false))
}

func TestRolloutRegression_ConfigMapRepeatedReconcileDoesNotChangeTemplate(t *testing.T) {
 assertStableTemplateAcrossReconciles(t, rolloutProvider(false, true))
}

func TestRolloutRegression_CombinedOutputsRepeatedReconcileDoesNotChangeTemplate(t *testing.T) {
 assertStableTemplateAcrossReconciles(t, rolloutProvider(true, true))
}

func assertStableTemplateAcrossReconciles(t *testing.T, provider resourcesv1alpha1.ResourceProvider) {
 t.Helper()
 obj := testDeployment(map[string]string{AnnotationEnabled:"true", AnnotationEnvPrefix:"CREDENTIALVERIFICATION"})
 providers := []resourcesv1alpha1.ResourceProvider{provider}
 if _, err := PatchWorkload(obj, providers, nil, nil); err != nil { t.Fatal(err) }
 expected := podTemplate(t, obj)
 expectedOrder := envOrder(t, obj)
 // Multiple iterations make Go's unspecified map traversal observable.
 for i:=0; i<100; i++ {
  if _, err := PatchWorkload(obj, providers, nil, nil); err != nil { t.Fatal(err) }
  if !reflect.DeepEqual(expected, podTemplate(t,obj)) {
   t.Fatalf("pod template changed on identical reconcile #%d (would cause a new ReplicaSet)\nfirst env order: %v\nnext env order:  %v", i+1, expectedOrder, envOrder(t,obj))
  }
 }
}

func TestRolloutRegression_FreshDeploymentsRenderIdentically(t *testing.T) {
 providers := []resourcesv1alpha1.ResourceProvider{rolloutProvider(true,true)}
 var first map[string]any
 var firstOrder []string
 for i:=0;i<100;i++ {
  obj:=testDeployment(map[string]string{AnnotationEnabled:"true"})
  if _,err:=PatchWorkload(obj,providers,nil,nil);err!=nil {t.Fatal(err)}
  if i==0 {first=podTemplate(t,obj);firstOrder=envOrder(t,obj);continue}
  if !reflect.DeepEqual(first,podTemplate(t,obj)) {
   t.Fatalf("identical inputs rendered different pod templates on run #%d\nfirst env order: %v\nnext env order:  %v",i+1,firstOrder,envOrder(t,obj))
  }
 }
}

func TestRolloutRegression_StableProviderHashDespiteChangedTemplate(t *testing.T) {
 obj:=testDeployment(map[string]string{AnnotationEnabled:"true"})
 providers:=[]resourcesv1alpha1.ResourceProvider{rolloutProvider(true,true)}
 if _,err:=PatchWorkload(obj,providers,nil,nil);err!=nil {t.Fatal(err)}
 baseline:=podTemplate(t,obj)
 initialAnnotations,_,_:=unstructured.NestedStringMap(obj.Object,"spec","template","metadata","annotations")
 initialHash:=initialAnnotations[AnnotationHash]
 for i:=0;i<100;i++ {
  if _,err:=PatchWorkload(obj,providers,nil,nil);err!=nil {t.Fatal(err)}
  annotations,_,_:=unstructured.NestedStringMap(obj.Object,"spec","template","metadata","annotations")
  if annotations[AnnotationHash]!=initialHash {t.Fatalf("provider hash unexpectedly changed")}
  if !reflect.DeepEqual(baseline,podTemplate(t,obj)) {
   t.Fatalf("same injector hash %q but different pod template after reconcile #%d; env order: %v",initialHash,i+1,envOrder(t,obj))
  }
 }
}
