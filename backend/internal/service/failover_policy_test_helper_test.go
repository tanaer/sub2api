package service

// testFailoverPolicy creates a FailoverPolicy with default codes for use in tests.
func testFailoverPolicy() *FailoverPolicy {
	p := &FailoverPolicy{}
	p.applyConfig(defaultFailoverCodes)
	return p
}
