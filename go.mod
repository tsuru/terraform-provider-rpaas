module github.com/tsuru/terraform-provider-rpaas

go 1.16

require (
	cloud.google.com/go/storage v1.16.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/ajg/form v1.5.1
	github.com/hashicorp/go-cty v1.4.1-0.20200414143053-d3edf31b6320
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.10.0
	github.com/hashicorp/yamux v0.0.0-20210707203944-259a57b3608c // indirect
	github.com/labstack/echo/v4 v4.6.1
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/sajari/fuzzy v1.0.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tsuru/commandmocker v0.0.0-20160909010208-e1d28f4f616a // indirect
	github.com/tsuru/config v0.0.0-20200717192526-2a9a0efe5f28 // indirect
	github.com/tsuru/gnuflag v0.0.0-20151217162021-86b8c1b864aa // indirect
	github.com/tsuru/rpaas-operator v0.27.8
	github.com/tsuru/tablecli v0.0.0-20180215113938-82de88f75181 // indirect
	github.com/tsuru/tsuru v0.0.0-20180820205921-0e7f7f02eac5
	istio.io/pkg v0.0.0-20210322140956-5892a3b28d3e
)

replace github.com/stern/stern => github.com/tsuru/stern v1.20.2-0.20210928180051-1157b938dc3f
