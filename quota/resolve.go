// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package quota

// resolve lets stdin win wherever both sources carry a bucket: it is free,
// synchronous and always current, while the API is cached and slightly
// behind. The API only contributes the model-scoped quotas, the credit
// balance and the buckets a host build did not send. A bucket missing from
// both stays missing: rendering it as 0 % would lie about the account.
func resolve(stdin, api Set) Set {
	merged := stdin
	if !merged.Session.IsValid() && api.Session.IsValid() {
		merged.Session = api.Session
	}
	if !merged.Weekly.IsValid() && api.Weekly.IsValid() {
		merged.Weekly = api.Weekly
	}
	merged.Scoped = api.Scoped
	merged.Extra = api.Extra
	return merged
}
