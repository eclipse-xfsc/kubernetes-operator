package injection

import (
 "reflect"
 "testing"

 resourcesv1alpha1 "github.com/eclipse-xfsc/kubernetes-operator/api/v1alpha1"
 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Regression: provider discovery order must not affect the rendered workload.
// These tests deliberately use multiple permutations, rather than relying on
// Go's randomized map traversal to trigger a failure.
func providerOrderFixtures() []resourcesv1alpha1.ResourceProvider {
 makeProvider := func(name string, env map[string]string) resourcesv1alpha1.ResourceProvider {
  p := resourcesv1alpha1.ResourceProvider{}
  p.Name = name
  p.Spec.Type = name
  p.Spec.Outputs.Env = env
  return p
 }
 return []resourcesv1alpha1.ResourceProvider{
  makeProvider("signer", map[string]string{"SIGNER_URL":"http://signer", "SIGNER_TOPIC":"signer-topic"}),
  makeProvider("did-resolver", map[string]string{"DID_RESOLVER":"http://resolver"}),
  makeProvider("nats", map[string]string{"NATS_HOST":"nats", "NATS_PORT":"4222"}),
  makeProvider("cassandra", map[string]string{"CASSANDRA_HOSTS":"cassandra", "CASSANDRA_PORT":"9042"}),
 }
}

func providerPermutations(p []resourcesv1alpha1.ResourceProvider) [][]resourcesv1alpha1.ResourceProvider {
 result := [][]resourcesv1alpha1.ResourceProvider{}
 var visit func(int)
 visit = func(i int) {
  if i == len(p) {
   result = append(result, append([]resourcesv1alpha1.ResourceProvider(nil), p...))
   return
  }
  for j := i; j < len(p); j++ {
   p[i], p[j] = p[j], p[i]
   visit(i+1)
   p[i], p[j] = p[j], p[i]
  }
 }
 visit(0)
 return result
}

func providerOrderAnnotations(t *testing.T, obj *unstructured.Unstructured) map[string]string {
 t.Helper()
 a, found, err := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
 if err != nil || !found { t.Fatalf("template annotations missing: %v", err) }
 return a
}

func renderProviderOrder(t *testing.T, providers []resourcesv1alpha1.ResourceProvider) *unstructured.Unstructured {
 t.Helper()
 obj := testDeployment(map[string]string{AnnotationEnabled:"true", AnnotationEnvPrefix:"CREDENTIALVERIFICATION"})
 if _, err := PatchWorkload(obj, providers, nil, nil); err != nil { t.Fatal(err) }
 return obj
}

func TestProviderOrderRegression_PermutationsProduceSamePodTemplate(t *testing.T) {
 permutations := providerPermutations(providerOrderFixtures())
 baseline := renderProviderOrder(t, permutations[0])
 expected := podTemplate(t, baseline)
 for i, providers := range permutations[1:] {
  got := renderProviderOrder(t, providers)
  if !reflect.DeepEqual(expected, podTemplate(t, got)) {
   t.Fatalf("provider permutation #%d changed pod template\nbaseline env: %v\nactual env:   %v", i+2, envOrder(t, baseline), envOrder(t, got))
  }
 }
}

func TestProviderOrderRegression_PermutationsProduceSameInjectorHash(t *testing.T) {
 permutations := providerPermutations(providerOrderFixtures())
 baseline := renderProviderOrder(t, permutations[0])
 expected := providerOrderAnnotations(t, baseline)[AnnotationHash]
 if expected == "" { t.Fatal("baseline injector hash is empty") }
 for i, providers := range permutations[1:] {
  got := renderProviderOrder(t, providers)
  actual := providerOrderAnnotations(t, got)[AnnotationHash]
  if actual != expected {
   t.Fatalf("provider permutation #%d changed injector hash: %s != %s", i+2, actual, expected)
  }
 }
}

func TestProviderOrderRegression_ReconcileWithReorderedProvidersIsIdempotent(t *testing.T) {
 permutations := providerPermutations(providerOrderFixtures())
 obj := renderProviderOrder(t, permutations[0])
 expected := podTemplate(t, obj)
 for i, providers := range permutations[1:] {
  if _, err := PatchWorkload(obj, providers, nil, nil); err != nil { t.Fatal(err) }
  if !reflect.DeepEqual(expected, podTemplate(t, obj)) {
   t.Fatalf("reconcile #%d with reordered providers changed pod template\nexpected env: %v\nactual env:   %v", i+2, envOrderFromTemplate(expected), envOrder(t, obj))
  }
 }
}

func envOrderFromTemplate(template map[string]any) []string {
 spec := template["spec"].(map[string]any)
 containers := spec["containers"].([]any)
 env := containers[0].(map[string]any)["env"].([]any)
 result := make([]string, 0, len(env))
 for _, e := range env { result = append(result, e.(map[string]any)["name"].(string)) }
 return result
}

func TestProviderOrderRegression_PermutationsPreserveManagedState(t *testing.T) {
 permutations := providerPermutations(providerOrderFixtures())
 baseline := renderProviderOrder(t, permutations[0])
 expected := ReadManagedState(baseline)
 for i, providers := range permutations[1:] {
  got := renderProviderOrder(t, providers)
  if !reflect.DeepEqual(expected, ReadManagedState(got)) {
   t.Fatalf("provider permutation #%d changed managed state", i+2)
  }
 }
}
