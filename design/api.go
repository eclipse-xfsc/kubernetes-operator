package design

import . "goa.design/goa/v3/dsl"

var _ = API("xsfc_resource_operator", func() {
	Title("XSFC Resource Operator API")
	Description("REST API for listing providers, consumers, injections, accounts, modules and monitored types.")
	Version("0.1.0")
	Server("operator", func() { Host("localhost", func() { URI("http://localhost:8088") }) })
})

var VersionResult = ResultType("application/vnd.xsfc.version", func() {
	Attributes(func() {
		Attribute("operatorVersion", String)
		Attribute("gitCommit", String)
		Attribute("buildDate", String)
	})
})

var ModuleResult = ResultType("application/vnd.xsfc.module", func() {
	Attributes(func() {
		Attribute("name", String)
		Attribute("version", String)
		Attribute("types", ArrayOf(String))
		Attribute("capabilities", ArrayOf(String))
	})
})

var ProviderResult = ResultType("application/vnd.xsfc.provider", func() {
	Attributes(func() {
		Attribute("type", String)
		Attribute("name", String)
		Attribute("namespace", String)
		Attribute("kind", String)
		Attribute("resource", String)
		Attribute("module", String)
	})
})

var ConsumerResult = ResultType("application/vnd.xsfc.consumer", func() {
	Attributes(func() {
		Attribute("type", String)
		Attribute("name", String)
		Attribute("namespace", String)
		Attribute("kind", String)
		Attribute("resource", String)
		Attribute("requestedTypes", ArrayOf(String))
	})
})

var InjectionResult = ResultType("application/vnd.xsfc.injection", func() {
	Attributes(func() {
		Attribute("consumerNamespace", String)
		Attribute("consumerName", String)
		Attribute("consumerKind", String)
		Attribute("container", String)
		Attribute("requestedTypes", ArrayOf(String))
		Attribute("mode", String)
		Attribute("status", String)
		Attribute("sourceManifest", String)
	})
})

var AccountResult = ResultType("application/vnd.xsfc.account", func() {
	Attributes(func() {
		Attribute("name", String)
		Attribute("namespace", String)
		Attribute("type", String)
		Attribute("consumerNamespace", String)
		Attribute("consumerName", String)
		Attribute("providerName", String)
		Attribute("providerNamespace", String)
		Attribute("createdBy", String)
	})
})

var ManifestResult = ResultType("application/vnd.xsfc.manifest", func() {
	Attributes(func() {
		Attribute("apiVersion", String)
		Attribute("kind", String)
		Attribute("name", String)
		Attribute("namespace", String)
		Attribute("requestedTypes", ArrayOf(String))
		Attribute("annotations", MapOf(String, String))
		Attribute("labels", MapOf(String, String))
	})
})

var _ = Service("inventory", func() {
	Method("version", func() { Result(VersionResult); HTTP(func() { GET("/version") }) })
	Method("modules", func() { Result(ArrayOf(ModuleResult)); HTTP(func() { GET("/modules") }) })
	Method("types", func() { Result(ArrayOf(String)); HTTP(func() { GET("/types") }) })
	Method("providers", func() { Result(ArrayOf(ProviderResult)); HTTP(func() { GET("/providers") }) })
	Method("consumers", func() { Result(ArrayOf(ConsumerResult)); HTTP(func() { GET("/consumers") }) })
	Method("injections", func() { Result(ArrayOf(InjectionResult)); HTTP(func() { GET("/injections") }) })
	Method("accounts", func() { Result(ArrayOf(AccountResult)); HTTP(func() { GET("/accounts") }) })
	Method("accountsByConsumer", func() {
		Payload(func() { Attribute("namespace", String); Attribute("name", String); Required("namespace", "name") })
		Result(ArrayOf(AccountResult))
		HTTP(func() { GET("/accounts/by-consumer/{namespace}/{name}"); Param("namespace"); Param("name") })
	})
	Method("manifestsRequestingInjection", func() { Result(ArrayOf(ManifestResult)); HTTP(func() { GET("/manifests/requesting-injection") }) })
	Method("healthz", func() { Result(String); HTTP(func() { GET("/healthz") }) })
	Method("readyz", func() { Result(String); HTTP(func() { GET("/readyz") }) })
})
