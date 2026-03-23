# Security Review: Wallet & Payments System

**Date:** 2026-03-23
**Scope:** wallet.b, wallet9p.b, x402.b, stripe.b, ethcrypto.b, ethrpc.b, wm/wallet.b, tools/wallet.b, tools/payfetch.b, nsconstruct.b (wallet gating), secp256k1.c, keccak256.c, keyring.c
**Reviewer:** Claude (automated)

---

## Executive Summary

The wallet and payments system has a **strong security architecture**. Key material is properly isolated in factotum, private keys are zeroed after use, namespace restrictions prevent agents from reading raw key data, and budget enforcement is server-side. The codebase shows evidence of deliberate security design, not bolt-on security.

However, I identified **3 high-severity, 4 medium-severity, and 7 low-severity findings** that should be addressed.

---

## Architecture Overview

```
                ┌──────────────────┐
                │   wm/wallet.b    │  GUI (human user)
                │   tools/wallet.b │  Agent tool (AI agent)
                │   tools/payfetch │  x402 auto-pay
                └────────┬─────────┘
                         │ 9P protocol
                ┌────────▼─────────┐
                │    wallet9p.b    │  9P file server
                │  (budget, sign,  │  (signing authority)
                │   pay, balance)  │
                └──┬──────────┬────┘
                   │          │
          ┌────────▼──┐   ┌──▼──────────┐
          │ factotum  │   │  ethrpc.b   │
          │ (key mgmt)│   │ (JSON-RPC)  │
          └───────────┘   └─────────────┘
```

**Key design strengths:**
- Private keys never leave factotum; wallet.b retrieves, signs, and zeros in a single function
- Agents interact via 9P filesystem — no direct factotum access
- Namespace restriction (nsconstruct.b) blocks agent access to `/mnt/factotum/ctl`
- Budget enforcement is server-side in wallet9p, not client-side
- secp256k1 uses constant-time Montgomery ladder and RFC 6979 deterministic k

---

## Findings

### HIGH SEVERITY

#### H1: JSON Injection in ethrpc.b RPC Parameters

**File:** `appl/lib/ethrpc.b:87,116,131,185`
**Impact:** An attacker who controls an address string or contract address could inject arbitrary JSON into the RPC request body.

The `getbalance`, `getnonce`, `sendrawtx`, and `ethcall` functions construct JSON-RPC params by string concatenation without sanitizing inputs:

```limbo
# ethrpc.b:87
params := "[\"" + addr + "\",\"latest\"]";

# ethrpc.b:185
params := "[{\"to\":\"" + contract + "\",\"data\":\"" + calldata + "\"},\"latest\"]";
```

If `addr` contains `"`, it breaks out of the JSON string. For example, an address like `0x1234\",\"injected\":\"` would corrupt the JSON-RPC request.

**Practical risk:** Medium-high. While Ethereum addresses are normally hex-only, the values come from user input paths (wm/wallet import, wallet9p writes). A malicious actor could craft a "recipient" address in a pay command to inject extra RPC parameters.

**Recommendation:** Validate that all address inputs match `/^0x[0-9a-fA-F]{40}$/` before interpolating into JSON. Alternatively, use the JSON library to construct params programmatically.

---

#### H2: Unchecked Budget Enforcement in payfetch.b

**File:** `appl/veltro/tools/payfetch.b:237-245`
**Impact:** The payfetch tool's `checkwalletbudget()` function only checks that the account exists — it does NOT actually verify the budget limit.

```limbo
checkwalletbudget(acctname: string, amount: string): string
{
    # Write to the wallet account ctl to check budget
    # For now, just verify the account exists
    addr := readfile("/n/wallet/" + acctname + "/address");
    if(addr == nil)
        return "account '" + acctname + "' not found";
    return nil;  # ← ALWAYS returns nil (OK) regardless of budget
}
```

The wallet.b library has proper `checkbudget()` / `recordspend()` functions, and wallet9p has budget support, but payfetch — the primary automated payment tool used by AI agents — **completely bypasses budget enforcement**.

**Practical risk:** HIGH. An AI agent using payfetch can spend unlimited funds, even when a budget has been explicitly set on the account. This defeats the purpose of the budget system for x402 payments.

**Recommendation:** Read the account budget via `/n/wallet/{name}/ctl`, parse it, compare against `amount`, and fail if over budget. After successful payment, write the spend to `/n/wallet/{name}/ctl` or use the existing `recordspend` mechanism.

---

#### H3: Path Traversal in wallet9p Account Names

**File:** `appl/veltro/wallet9p.b:594-644` (Qnew write handler)
**Impact:** Account names are not validated for dangerous characters. A name like `../ctl` or `../../mnt/factotum` could potentially cause path traversal in the 9P navigator.

The `Qnew` write handler passes the account name directly from user input to `wallet->createaccount()` and into the accounts list, which becomes part of the 9P namespace. While the styx navigator does name matching (not path-based lookup), the factotum service key is constructed as:

```limbo
# wallet.b:370
servicekey(name: string, accttype: int): string
{
    if(accttype == ACCT_ETH)
        return "wallet-eth-" + name;
```

And the factotum command includes unvalidated content:
```limbo
# wallet.b:384
cmd := "key proto=pass service=" + svc + " user=key !password=" + hexkey;
```

If `name` contains spaces or `!` characters, it could inject additional factotum attributes.

**Practical risk:** High for the factotum attribute injection vector. A name like `myaccount user=admin !password=stolen` would add malformed key attributes.

**Recommendation:** Validate account names to allow only `[a-zA-Z0-9_-]` and reject names containing `/`, `..`, spaces, `!`, `=`, or other special characters. Apply validation in both `wallet9p.b` (Qnew handler) and `wallet.b` (createaccount/importaccount).

---

### MEDIUM SEVERITY

#### M1: Stripe API Key Persisted in Module-Level Variable

**File:** `appl/lib/stripe.b:38,67`
**Impact:** The Stripe secret key is stored in a module-global variable `secretkey` for the lifetime of the process.

```limbo
secretkey: string;
# ...
secretkey = apikey;
```

Unlike ETH private keys which are retrieved-and-zeroed per operation, the Stripe API key persists in memory after `init()`. In Limbo, strings are garbage-collected but there is no explicit zeroing guarantee. If the Dis VM process memory is inspectable (e.g., via `/prog/*/heap` or core dump), the Stripe key would be recoverable.

**Recommendation:** Retrieve the Stripe API key from factotum on each API call, matching the ETH pattern. Alternatively, document this as an accepted risk since the Stripe module is only loaded on-demand.

---

#### M2: No TLS Certificate Validation for RPC Endpoints

**File:** `appl/lib/ethrpc.b:313`
**Impact:** The webclient library is used to make HTTPS requests to RPC endpoints. If the webclient does not perform certificate validation (or if the user configures an HTTP endpoint via the `rpc` ctl command), JSON-RPC calls — including signed transaction submissions — could be intercepted.

The `ctl` command `rpc <url>` (wallet9p.b:563) allows setting an arbitrary RPC URL with no validation:

```limbo
} else if(cmd == "rpc" && ntoks >= 2) {
    ethrpc->setrpc(hd tl toks);
```

An agent or compromised process with access to `/n/wallet/ctl` could redirect RPC to a malicious endpoint to:
1. Return fake balance data
2. Capture signed transactions and front-run them
3. Return fake gas prices to drain funds via excessive fees

**Recommendation:** Validate that RPC URLs use HTTPS. Consider maintaining a hard-coded allowlist of trusted RPC endpoints and requiring explicit user confirmation to add new ones.

---

#### M3: Race Condition in Budget Enforcement

**File:** `appl/lib/wallet.b:325-362`
**Impact:** Budget check and spend recording are separate operations with no locking:

```limbo
checkbudget(acct: ref Account, amount: big): string  # CHECK
# ... time passes, payment executes ...
recordspend(acct: ref Account, amount: big)           # RECORD
```

In wallet9p.b, the pay handler calls these sequentially but there is no mutex. Two concurrent payment requests to the same account could both pass `checkbudget` before either calls `recordspend`, allowing the total spend to exceed `maxpersess`.

**Practical risk:** Medium. Requires concurrent access to the same account through different fids, which is possible if multiple agent tools or payfetch instances run simultaneously.

**Recommendation:** Use a channel-based lock (mutex) around the check-sign-record sequence in wallet9p's pay handler, or combine check+record into an atomic `reservebudget()` function.

---

#### M4: SSRF Protection Incomplete in payfetch.b

**File:** `appl/veltro/tools/payfetch.b:357-368`
**Impact:** The SSRF blocklist is incomplete:

```limbo
isblocked(host: string): int
{
    if(host == "localhost" || host == "127.0.0.1")
        return 0;   # ← localhost explicitly ALLOWED
    if(host == "::1" || host == "0.0.0.0")
        return 1;
    if(hasprefix(host, "10.") || hasprefix(host, "192.168.") ||
       hasprefix(host, "169.254."))
        return 1;
    return 0;
}
```

Missing from the blocklist:
- **172.16.0.0/12** private range (172.16.x.x through 172.31.x.x)
- **100.64.0.0/10** (CGNAT / cloud metadata in some providers)
- **fd00::/8** IPv6 ULA
- **[::ffff:127.0.0.1]** IPv4-mapped IPv6
- DNS rebinding attacks (hostname resolves to internal IP)

Additionally, `localhost` and `127.0.0.1` are explicitly allowed, which is intentional for development but dangerous in production — a malicious x402 server could redirect the agent to `http://localhost:PORT/internal-endpoint`.

**Recommendation:** Block the 172.16/12 range. Consider making localhost allowance configurable or production-only blocked. Add a comment documenting the DNS rebinding risk.

---

### LOW SEVERITY

#### L1: Private Key Displayed in GUI Import Form (Briefly)

**File:** `appl/wm/wallet.b:475,961`
**Impact:** The import form correctly uses a masked text field (`Textfield.mk(..., 1)` = masked). However, the hex key is then sent unmasked over the 9P protocol:

```limbo
cmd := "import eth " + chain + " " + name + " " + hexkey;
n := writewalletctl("new", cmd);
```

The key traverses the 9P wire protocol in cleartext to wallet9p. Since wallet9p is mounted locally (same process group), the risk is limited to in-memory visibility. The key is also held in `f_key.value()` as an unzeroed Limbo string until garbage collection.

**Recommendation:** After calling `writewalletctl`, explicitly clear `f_key` value. Document that 9P transport is local-only.

---

#### L2: Integer Overflow in strtobig/strtoint

**File:** `appl/veltro/wallet9p.b:1229-1240,1290-1301`
**Impact:** `strtobig()` and `strtoint()` parse decimal strings without overflow checking. For `strtoint()`, values > 2^31-1 silently overflow. For `strtobig()`, values > 2^63-1 silently overflow.

Used in payment amount parsing (`executepayment`, `executeerc20`, `executestripepayment`), this could cause:
- A very large amount string to wrap to a small value, sending less than intended
- A crafted amount to overflow to a negative or zero value, bypassing budget checks

**Recommendation:** Add overflow detection or range checks for payment amounts.

---

#### L3: No Gas Limit Validation for ERC-20 Payments

**File:** `appl/veltro/wallet9p.b:1181`
**Impact:** The ERC-20 gas limit is hardcoded to 100,000:

```limbo
gaslimit := big 100000;  # ERC-20 transfers need ~65000 gas
```

If the gas price is at the 100 gwei cap and gas limit is 100,000, the maximum gas fee is 0.01 ETH (~$30 at current prices). This is reasonable but not configurable. More importantly, if a token contract requires more than 100,000 gas (e.g., proxy contracts with complex logic), the transaction will revert and the gas fee is lost.

**Recommendation:** Consider using `eth_estimateGas` to set an accurate limit, with the hardcoded value as a safety cap.

---

#### L4: History File Not Bounded

**File:** `appl/veltro/wallet9p.b:728`
**Impact:** Transaction history is appended to an in-memory list with no size limit:

```limbo
as.history = ("pay " + payamt + " " + payrecip + " " + txhash) :: as.history;
```

Over a long session with many transactions, this list grows unboundedly.

**Recommendation:** Cap history to the most recent N entries (e.g., 100).

---

#### L5: No Replay Protection on x402 Nonces

**File:** `appl/lib/x402.b:178-181`
**Impact:** The x402 nonce is generated from `/dev/random` which is good, but there's no record of used nonces. While EIP-3009 nonces are enforced on-chain (the token contract tracks used nonces), if the same nonce were somehow reused before on-chain settlement, a facilitator could replay the authorization.

**Practical risk:** Very low — the 32-byte random nonce has negligible collision probability, and on-chain enforcement provides the real protection.

**Recommendation:** No action needed; document the on-chain enforcement as the primary defense.

---

#### L6: Pending Payments List Unbounded

**File:** `appl/veltro/wallet9p.b:106,711`
**Impact:** Pending payments (`pendingpays`) grow without cleanup. Denied or completed entries remain in the list forever, leaking memory and making `/n/wallet/pending` reads progressively larger.

**Recommendation:** Remove resolved entries after a timeout or on explicit cleanup.

---

#### L7: Error Messages May Leak Internal Paths

**File:** Various
**Impact:** Error messages include internal filesystem paths (e.g., `/mnt/factotum/ctl`, `/n/wallet/...`) which could leak system architecture to agent tools. Low risk since the namespace is already documented, but follows defense-in-depth.

**Recommendation:** Use generic error messages for agent-facing interfaces.

---

## Positive Security Properties

These are design strengths worth preserving:

| Property | Implementation | Location |
|----------|---------------|----------|
| **Key isolation** | Private keys retrieved, used, and zeroed in single functions | wallet.b:268-306 |
| **Factotum separation** | Keys stored in Plan 9 factotum, never in application memory | wallet.b:379-395 |
| **Namespace gating** | Agents get `/n/wallet` only if explicitly granted in caps.paths | nsconstruct.b:225-229 |
| **Factotum blocked** | Agents cannot read `/mnt/factotum/ctl` (not in any allowlist) | nsconstruct.b architecture |
| **Server-side signing** | Agents write hashes, wallet9p signs internally | wallet9p.b:656-675 |
| **Constant-time crypto** | secp256k1 Montgomery ladder, no secret-dependent branches | secp256k1.c |
| **RFC 6979 deterministic k** | Eliminates k-reuse attacks entirely | secp256k1.c |
| **Low-S normalization** | BIP-62/EIP-2 compliance prevents malleability | secp256k1.c |
| **Gas price cap** | 100 gwei cap prevents fee spike exploitation | wallet9p.b:1068 |
| **Masked import field** | Private key input is masked in GUI | wm/wallet.b:475 |
| **SSRF protection** | payfetch blocks private network ranges | payfetch.b:357-368 |
| **Secstore encryption** | AES-256-GCM for persistent key storage | secstore.m |
| **EIP-155 replay protection** | Chain ID included in transaction signing | ethcrypto.b:279 |
| **EIP-55 checksummed addresses** | Prevents address typo attacks | ethcrypto.b:59-91 |
| **Debounced factotum sync** | Prevents PAK handshake storms on batch imports | wallet9p.b:846-895 |

---

## Recommendations Summary

| ID | Severity | Effort | Description |
|----|----------|--------|-------------|
| H1 | High | Low | Validate addresses before JSON interpolation in ethrpc.b |
| H2 | High | Low | Implement actual budget checking in payfetch.b |
| H3 | High | Low | Validate account names against `[a-zA-Z0-9_-]` |
| M1 | Medium | Low | Zero Stripe API key or retrieve per-call |
| M2 | Medium | Medium | Validate RPC URLs, require HTTPS, consider allowlist |
| M3 | Medium | Medium | Add mutex around budget check-sign-record sequence |
| M4 | Medium | Low | Add 172.16/12 to SSRF blocklist |
| L1 | Low | Low | Clear key field value after import |
| L2 | Low | Low | Add overflow checks in strtobig/strtoint |
| L3 | Low | Medium | Use eth_estimateGas for ERC-20 transfers |
| L4 | Low | Low | Cap history list size |
| L5 | Low | None | Document on-chain nonce enforcement |
| L6 | Low | Low | Cleanup resolved pending payments |
| L7 | Low | Low | Sanitize error messages for agent-facing APIs |

---

## Scope Exclusions

- **secp256k1.c deep cryptanalysis**: The C implementation follows standard constant-time patterns with Montgomery ladder and RFC 6979. A full formal verification is outside scope but the code structure is sound.
- **Secstore protocol**: The PAK authentication protocol is inherited from Plan 9 and is outside the scope of this wallet-specific review.
- **Webclient TLS implementation**: The TLS stack itself was not reviewed; only its usage in the payment system.
- **Dis VM memory safety**: The Dis VM provides memory safety guarantees; this review focuses on logic-level vulnerabilities.
