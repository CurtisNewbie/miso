# Nacos config-log masking covers credential keywords

Status: ready-for-agent

## Problem

miso masks secret values in Nacos config bodies before logging them, with a keyword list built in
`init()` of `middleware/nacos/nacos.go`. The list misses five credential words that config keys in
the field use: `pwd`, `passwd`, `credential`, `authorization`, `pem`.

Adding the words alone does not fix it. The pattern is case sensitive and only matches a keyword
that ends the key, so two shapes still print in clear:

- a key that spells the word in mixed case, such as `appleMerchantIdPwd`;
- a key that carries more characters after the word, such as `pemContent`, or the plural form of a
  word that is singular in the list.

## Decision

Extend the keyword list, and change how a keyword is matched so the new words do their job.

- Append `pwd`, `passwd`, `credential`, `authorization`, `pem` to `desensitizeKw`. The
  `strutil.CamelCase` loop stays as it is.
- Compile the pattern case insensitively (`(?i)`).
- Let a keyword sit anywhere inside the key, bounded by key characters:
  `([A-Za-z0-9_.\-]*(kw)[A-Za-z0-9_.\-]*)`.
- Keep the value side `("?[^\n"]+"?)` and replace it with `***`, so the key, the separator and the
  indentation survive in the log.

## Contract and invariants

- `desensitizeConfigContent(s)` masks the value of any line whose key contains a keyword, ignoring
  case.
- Key, separator, quote characters and indentation stay readable. Only the value becomes `***`.
- A line with no keyword in its key comes back unchanged.
- The pattern compiles once in `init()`; matching costs nothing extra per call.
- Over-masking is accepted. `key` and `token` now match inside any key, so a line such as
  `hotkey: 1` also loses its value. A masked non-secret is a cosmetic log defect; an unmasked
  secret is a leak.

## Acceptance criteria

- [ ] `middleware/nacos/nacos_test.go` (package `nacos`, so it can reach the unexported
      `desensitizeConfigContent`) is a table test covering, for the five new keywords, the plain
      lowercase key, the mixed-case key, and the key with the word followed by more characters.
- [ ] The control key `username` comes back unchanged.
- [ ] Existing keywords still mask: `secret-key`, `access-token`, `password`.
- [ ] `go test ./middleware/nacos/` passes.
- [ ] `go build ./...` passes.

## Out of scope

- Multi-line PEM and keystore bodies, and YAML block scalars (`|`, `>`). Only the first line of such
  a value is masked. Masking the rest of the block is a separate change.
- Values nested under a `secret:` map whose keys are access-key ids. Key-name matching cannot see
  them.
- Comment handling. The pattern matches after `#`, so a commented-out key is already masked.
- Preserving the quote characters around a masked value; they are part of the value side today.