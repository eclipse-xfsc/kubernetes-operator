package modules

import (
    "strings"
    "sigs.k8s.io/controller-runtime/pkg/client"
)
const (
    ProviderLabel="xfsc.io/resource-provider"
    ResourceTypeLabel="xfsc.io/resource-type"
    ResourceNameLabel="xfsc.io/resource-name"
    InjectEnabledAnno="inject.xfsc.io/enabled"
    InjectTypesAnno="inject.xfsc.io/types"
    InjectModeAnno="inject.xfsc.io/mode"
)
func IsProvider(obj client.Object, typ string) bool { l:=obj.GetLabels(); return l[ProviderLabel]=="true" && l[ResourceTypeLabel]==typ }
func ProviderName(obj client.Object) string { if v:=obj.GetLabels()[ResourceNameLabel]; v!="" { return v }; return "default" }
func RequestedTypes(obj client.Object) []string { a:=obj.GetAnnotations(); if a[InjectEnabledAnno]!="true" { return nil }; raw:=a[InjectTypesAnno]; if raw=="" { return nil }; ps:=strings.Split(raw, ","); out:=[]string{}; for _,p:=range ps { p=strings.TrimSpace(p); if p!="" { out=append(out,p) } }; return out }
func WantsType(obj client.Object, typ string) bool { for _,t:=range RequestedTypes(obj) { if t==typ { return true } }; return false }
