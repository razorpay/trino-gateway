# Cross-Border Payment Patterns

Comprehensive reference for cross-border payment checks covering CFB fee handling, currency validation, DCC/MCC lifecycle, wallet payments, skip3DS, recurring payments, network tokenization, and scrooge refunds. Used by the pre-mortem skill's Service Integration agent (Agent 2) when PR diff touches cross-border paths or forex/currency/DCC/MCC/CFB keywords.

## Table of Contents

1. [CFB Fee Handling Patterns](#cfb-fee-handling-patterns) — Fee subtraction, conversion rates, recalculation
2. [Currency Validation Patterns](#currency-validation-patterns) — Currency consistency, INR-INR, MCC vs DCC
3. [Exchange Rate Patterns](#exchange-rate-patterns) — Markdown/markup, denomination, rounding
4. [Lifecycle Transitions](#lifecycle-transitions) — Auth→capture fee conversion, base amount recalculation
5. [Payments-Card Cross-Border Patterns](#payments-card-cross-border-patterns) — DCC blacklist, forex_applied, EEFC
6. [MCC/DCC Lifecycle Flows](#mccdcc-lifecycle-flows-and-currency-states) — End-to-end flows, currency states per stage
7. [Scrooge Refund Patterns](#cross-border-patterns-in-scrooge-refunds-service) — Math.Floor, 3-decimal rounding, LRS
8. [Wallet Payment Patterns](#wallet-payments-apple-pay--google-pay-cross-border-patterns) — DCC caching, risk tokens
9. [Skip3DS Patterns](#skip3ds-cross-border-patterns) — Rule engine, authorization retry, merchant risk profiles
10. [Recurring Payment Patterns](#cross-border-recurring-payments-patterns) — forex_applied validation, max_amount, token service
11. [Network Tokenization Patterns](#network-tokenization-cross-border-patterns) — Routing, smart retry exclusion, Amex
12. [Common Antipatterns](#common-anti-patterns) — Quick reference for frequent mistakes

---

# CFB Fee Handling Patterns

## Why Fee Handling Matters

CFB (Customer Fee Bearer) means the customer pays merchant fees on top of the transaction amount. For cross-border payments with currency conversion, incorrect fee handling causes:

- **Merchant settlement amount errors** (too high or too low)
- **Contract violations** (merchant receives markdown benefit on fee)
- **Failed validations** (mixed currency comparisons)
- **Customer overcharges** (fee applied twice)

These issues have caused production incidents affecting international merchants and revenue.

## Critical Pattern 1: Fee Subtraction Before Markdown

### The Rule

**ALWAYS subtract fee from amount BEFORE applying markdown calculation.**

### Why This Matters

- Markdown is merchant benefit (2% favorable rate)
- Merchant should receive markdown on payment amount ONLY
- Fee is charged separately to customer
- If fee not subtracted, merchant gets markdown benefit on fee (incorrect)

### Wrong vs Correct

❌ **WRONG**:
```go
// Fee not subtracted - merchant gets markdown on full amount including fee
markdownExchangeRate := exchangeRate - exchangeRate*markdownPercent/100
baseAmount := math.Ceil(markdownExchangeRate * amount * denominationFactor)
baseFee := math.Ceil(exchangeRate * fee * denominationFactor)
total := baseAmount + baseFee  // WRONG: Double-counts markdown benefit
```

✅ **CORRECT**:
```go
// Subtract fee FIRST
amount -= fee
markdownExchangeRate := exchangeRate - exchangeRate*markdownPercent/100
baseAmount := math.Ceil(markdownExchangeRate * amount * denominationFactor)
baseFee := math.Ceil(exchangeRate * fee * denominationFactor)
total := baseAmount + baseFee  // CORRECT
```

### Real Example

```
Payment: 1000 INR, Fee: 50 INR, Base: USD, Rate: 83.00, Markdown: 2%
Markdown Rate: 83.00 - (83.00 * 0.02) = 81.34

❌ WRONG (fee not subtracted):
Base Amount: 1000 / 81.34 = 12.29 USD
Base Fee: 50 / 83.00 = 0.60 USD
Total: 12.89 USD (merchant receives too much!)

✅ CORRECT (fee subtracted first):
Base Amount: (1000 - 50) / 81.34 = 11.68 USD
Base Fee: 50 / 83.00 = 0.60 USD
Total: 12.28 USD (correct)
```

**Impact**: In the wrong scenario, merchant receives $0.61 extra per transaction. At scale, this compounds into significant revenue leakage.

## Critical Pattern 2: Fee Conversion Rate

### The Rule

**Fee must be converted at original exchangeRate, NOT markdownExchangeRate.**

### Why This Matters

- Markdown is merchant benefit (only applies to payment amount)
- Fee conversion should be transparent (no markdown benefit)
- Customer sees exact fee amount converted at standard rate

### Wrong vs Correct

❌ **WRONG**:
```go
baseFee := math.Ceil(markdownExchangeRate * fee * denominationFactor)
// Fee gets markdown benefit - incorrect
```

✅ **CORRECT**:
```go
baseFee := math.Ceil(exchangeRate * fee * denominationFactor)
// Fee at original rate - correct
```

### Example

```
Fee: 200 cents (2 USD)
Exchange Rate: 83.00 INR/USD
Markdown Rate: 81.34 INR/USD

❌ WRONG (markdown rate used):
baseFee = 200 / 100 * 81.34 = 162.68 INR (~1.96 USD)
Customer sees: 1.96 USD fee (confusing - they were charged 2 USD!)

✅ CORRECT (original rate used):
baseFee = 200 / 100 * 83.00 = 166 INR (2.00 USD)
Customer sees: 2.00 USD fee (transparent - matches charge)
```

## Critical Pattern 3: Fee Recalculation on Update

### The Rule

**When fee is provided in update flow, base_amount MUST be recalculated.**

### Why This Matters

- Initial forex_charges may be created without fee (preview flow)
- Actual payment includes fee
- Using cached base_amount with new fee gives wrong settlement

### Code Location

`/payments-cross-border/internal/forex_processing/forex_charges/core.go:178-184`

### Correct Pattern

```go
if !utils.IsEmpty(req.Fee) {
    // Re-calculate base amount with fee
    baseAmount, baseFee := calculateBaseAmountAndFee(
        amount, req.Fee, baseCurrency,
        forexChargesEntity.GetBaseCurrency(),
        forexChargesEntity.GetBaseForexRate(),
        forexChargesEntity.GetMarkdownPercent(),
    )
    forexChargesEntity.SetBaseAmount(baseAmount)
    forexChargesEntity.SetBaseFee(baseFee)
}
```

### Flow Example

```
Step 1: Create forex_charges (currency picker preview)
- amount: 10000 cents ($100)
- fee: 0 (not yet known)
- baseAmount calculated: $100 converted to SGD = 135 SGD

Step 2: Update with actual fee
- fee: 200 cents ($2)
- MUST recalculate: baseAmount = ($100 - $2) converted = 132.30 SGD

❌ WRONG: Keep cached baseAmount (135 SGD) + add baseFee
✅ CORRECT: Recalculate baseAmount (132.30 SGD) + baseFee
```

## Test Cases to Verify Correctness

### Test 1: CFB Fee Subtraction

```go
func TestCalculateBaseAmount_CFBFeeSubtraction(t *testing.T) {
    amount := int64(100000)  // 1000 USD in cents
    fee := int64(200)        // 2 USD fee
    rate := 83.0
    markdown := 2.0

    // Expected calculation
    expectedAmount := amount - fee  // 99800
    markdownRate := rate - (rate * markdown / 100)  // 81.34
    expectedBase := math.Ceil(float64(expectedAmount) / 100 * markdownRate)

    actualBase := calculateBaseAmount(amount, fee, rate, markdown)

    assert.Equal(t, expectedBase, actualBase, "Fee must be subtracted before markdown")
}
```

### Test 2: Fee at Original Rate

```go
func TestCalculateBaseFee_OriginalRate(t *testing.T) {
    fee := int64(200)
    exchangeRate := 83.0
    markdownRate := 81.34

    // baseFee should use exchangeRate, NOT markdownRate
    expectedBaseFee := math.Ceil(float64(fee) / 100 * exchangeRate)

    actualBaseFee := calculateBaseFee(fee, exchangeRate, markdownRate)

    assert.Equal(t, expectedBaseFee, actualBaseFee, "Fee must use original rate")
}
```

### Test 3: Update Flow Recalculation

```go
func TestUpdateForexCharges_FeeRecalculation(t *testing.T) {
    // Initial: no fee
    entity := createForexCharges(10000, "USD", "SGD")
    initialBaseAmount := entity.GetBaseAmount()

    // Update: fee provided
    req := &UpdateRequest{Fee: 200}
    UpdateForexCharges(entity, req)

    updatedBaseAmount := entity.GetBaseAmount()

    // Base amount must be different (recalculated with fee subtracted)
    assert.NotEqual(t, initialBaseAmount, updatedBaseAmount,
        "Base amount must be recalculated when fee added")
    assert.Less(t, updatedBaseAmount, initialBaseAmount,
        "Base amount should decrease when fee subtracted")
}
```

## Common Mistakes and Fixes

### Mistake 1: Fee Added After Markdown

```go
❌ WRONG
baseAmount := markdownRate * amount
total := baseAmount + fee  // Adding fee after conversion
```

**Fix**: Subtract fee BEFORE conversion
```go
✅ CORRECT
amount -= fee
baseAmount := markdownRate * amount
total := baseAmount + convertedFee
```

### Mistake 2: Fee Gets Markdown Benefit

```go
❌ WRONG
baseFee := markdownRate * fee
```

**Fix**: Use original exchangeRate for fee
```go
✅ CORRECT
baseFee := exchangeRate * fee
```

### Mistake 3: Cached Base Amount with New Fee

```go
❌ WRONG
if req.Fee != nil {
    entity.SetFee(req.Fee)  // Only updating fee field
}
```

**Fix**: Recalculate baseAmount when fee changes
```go
✅ CORRECT
if req.Fee != nil {
    baseAmount, baseFee := calculateBaseAmountAndFee(...)
    entity.SetBaseAmount(baseAmount)
    entity.SetBaseFee(baseFee)
}
```

## Source References

- **Core calculation**: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:699-715`
- **Update flow**: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:178-184`
- **Production incident**: Cross-border brainstorm document (2026-02-18)

## Related Patterns

- [Currency Validation](#currency-validation-patterns) - Ensuring currencies match before operations
- [Exchange Rate Patterns](#exchange-rate-patterns) - Markdown/markup application
- [Lifecycle Transitions](#lifecycle-transitions) - Fee currency changes at capture


---

# Currency Validation Patterns

## Currency Consistency Rules

### Rule 1: Match Currencies Before Arithmetic

**Never perform arithmetic on amounts in different currencies without conversion.**

❌ **WRONG**:
```php
// $payment->amount in USD, $payment->fee in INR
if ($payment->getAmount() - $payment->getFee() < $order->getAmount()) {
    throw new ValidationException();
}
```

**Why Wrong**: Subtracting INR fee from USD amount produces meaningless result. For example, $100 - ₹50 = nonsense.

✅ **CORRECT**:
```php
// Convert fee to payment currency first
$feeInPaymentCurrency = $this->convertFee(
    $payment->getFee(),
    Currency::INR,
    $payment->getCurrency()
);

if ($payment->getAmount() - $feeInPaymentCurrency < $order->getAmount()) {
    throw new ValidationException();
}
```

### Rule 2: Document Currency at Each Stage

**Fee currency changes across lifecycle stages - must be explicit.**

Lifecycle stages:
1. **Authorization** - Fee in payment currency
2. **Capture** - Fee in base currency (INR for Indian merchants)
3. **Settlement** - Fee in merchant base currency

Example:
```php
// Auth stage
$payment->fee = 200;  // 2 USD (payment currency)
$payment->currency = Currency::USD;

// Capture stage (MCC flow)
if ($payment->getCurrency() !== Currency::INR
    and $payment->isFeeBearerCustomer()
    and $this->isMCCAppliedPayment($payment)) {
    // Convert fee from payment currency to INR
    $payment->setFee($txn->getFee());  // Now in INR
}
```

### Rule 3: INR-INR Special Case

**When base_currency == payment_currency == INR, no conversion needed.**

Code location: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:195-200`

```go
if getMerchantBaseCurrency(req.MerchantCountry) == forexChargesEntity.GetCurrency() &&
    amount != forexChargesEntity.GetBaseAmount() &&
    forexChargesEntity.GetPurpose() != common.PurposeImportFlow {
    // For INR → INR, base_amount should equal amount
    forexChargesEntity.SetBaseAmount(amount)
}
```

**Why**: No forex conversion for same-currency transactions, even if someone sends wrong amount.

## When to Convert Currencies

### Scenario 1: MCC CFB Payment

**Flow**: USD payment → SGD merchant (Singapore)

```
Payment Amount: 100 USD
Fee: 2 USD (in payment currency at auth)
Base Currency: SGD
Exchange Rate: 1.35 SGD/USD
Markdown: 2%

Auth Stage:
- payment.amount = 100 USD
- payment.fee = 2 USD
- payment.currency = USD

Capture Stage:
- Convert to base currency (SGD)
- Apply markdown to amount only
- Fee at original rate
- Result: baseAmount + baseFee in SGD
```

### Scenario 2: DCC Payment

**Flow**: International card → INR merchant (India)

```
Customer sees: 100 EUR
Charged to card: 100 EUR
Merchant receives: INR equivalent (with markup)

Markup: 8% (customer sees 2.4% due to display factor)
Exchange Rate: 90 INR/EUR
Markup Rate: 90 + (90 * 0.08) = 97.2 INR/EUR

Settlement: 100 EUR * 97.2 = 9720 INR
```

### Scenario 3: LRS Transaction

**Flow**: Outward remittance from India

```
All amounts in INR (simplified):
- payment.currency = INR
- payment.amount = amount in INR
- fee in INR
- No conversion needed

LRS specific: Always works in INR regardless of destination currency
```

## MCC vs DCC Differences

| Aspect | DCC | MCC |
|--------|-----|-----|
| **Direction** | Inward (customer pays in native) | Outward (merchant settles in native) |
| **Rate** | Markup (8%) | Markdown (2%) |
| **Who Benefits** | Merchant (higher revenue) | Merchant (favorable settlement) |
| **CFB Fee** | In payment currency | Converted at capture |
| **Use Case** | International cards to Indian merchants | Indian merchants accepting foreign currency |

## Validation at Auth vs Capture

### Authorization Stage

**Validations**:
- Payment currency valid and supported
- Fee in payment currency (if CFB)
- Forex charges created with preview rates
- Amount matches order amount (same currency comparison)

**Code**:
```php
// Auth validation
if ($payment->getCurrency() !== $order->getCurrency()) {
    throw new CurrencyMismatchException();
}

// CFB fee in payment currency
if ($payment->isFeeBearerCustomer()) {
    $fee = $this->calculateFeeInCurrency($payment->getAmount(), $payment->getCurrency());
    $payment->setFee($fee);
}
```

### Capture Stage

**Conversions**:
- Fee converted from payment currency to base currency
- Base amount finalized with actual exchange rate
- Transaction entity stores everything in base currency

**Code**:
```php
// Capture conversion (MCC CFB)
if ($payment->getCurrency() !== Currency::INR
    and $payment->isFeeBearerCustomer()
    and $this->isMCCAppliedPayment($payment)) {

    // Fee now in INR (from txn entity)
    $payment->setFee($txn->getFee());

    // Amount converted to base currency
    $payment->setBaseAmount($txn->getBaseAmount());
}
```

## Common Validation Errors

### Error 1: Currency Mismatch in Callback

Semgrep rule: `/mozart/.github/semgrep-rules/mozart-security-02-check-currency-rule-sequence.yaml`

```go
// Payment currency must match callback currency
if callbackCurrency != payment.Currency {
    return errors.New("currency mismatch in callback")
}
```

### Error 2: Refund Currency Mismatch

Semgrep rule: `/mozart/.github/semgrep-rules/mozart-security-09_refund-currency-validation.yaml`

```
Refund currency must equal payment currency
```

### Error 3: Base Currency Assumption

```go
❌ WRONG - Assuming base currency from request
baseCurrency := req.BaseCurrency

✅ CORRECT - Derive from merchant country
baseCurrency := getMerchantBaseCurrency(req.MerchantCountry)
```

**Code location**: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:676-685`

```go
func getMerchantBaseCurrency(merchantCountry string) string {
    if merchantCountry == "MY" {
        return currency.MYR  // Malaysia
    }
    if merchantCountry == "SG" {
        return currency.SGD  // Singapore
    }
    return currency.INR  // Default: India
}
```

**Why**: Base currency determined by merchant's regulatory domicile, not request parameter.

## Test Cases

### Test 1: Currency Consistency

```go
func TestValidateCurrencyConsistency(t *testing.T) {
    payment := &Payment{Amount: 10000, Currency: "USD", Fee: 200}
    order := &Order{Amount: 10200, Currency: "USD"}

    // Should pass - same currency
    err := validateCurrencyConsistency(payment, order)
    assert.NoError(t, err)

    // Should fail - mixed currency
    order.Currency = "INR"
    err = validateCurrencyConsistency(payment, order)
    assert.Error(t, err, "currency mismatch")
}
```

### Test 2: Fee Conversion at Capture

```go
func TestFeeConversionAtCapture_MCC_CFB(t *testing.T) {
    payment := &Payment{
        Amount: 10000, Currency: "USD", Fee: 200,  // Fee in USD
        IsFeeBearerCustomer: true,
    }

    // Capture with MCC
    capture(payment)

    // Assert: Fee now in INR
    assert.Equal(t, "INR", payment.FeeCurrency)
    assert.Greater(t, payment.Fee, int64(200))  // Converted to INR (higher value)
}
```

### Test 3: INR-INR No Conversion

```go
func TestINRtoINR_NoConversion(t *testing.T) {
    req := &CreateForexChargesRequest{
        Amount: 100000,
        Currency: "INR",
        MerchantCountry: "IN",  // India
    }

    charges := createForexCharges(req)

    // Assert: base_amount == amount (no conversion)
    assert.Equal(t, req.Amount, charges.BaseAmount)
}
```

## Source References

- **API fee conversion**: `/api/app/Models/Payment/Processor/Capture.php`
- **Base currency derivation**: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:676-685`
- **INR-INR handling**: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:195-200`
- **Semgrep rules**: `/mozart/.github/semgrep-rules/mozart-security-*.yaml`

## Related Patterns

- [CFB Fee Handling](#cfb-fee-handling-patterns) - Fee subtraction and conversion
- [Exchange Rate Patterns](#exchange-rate-patterns) - Markup/markdown application
- [Lifecycle Transitions](#lifecycle-transitions) - Currency changes across stages


---

# Exchange Rate Patterns

## Markdown vs Markup Application

### Markdown (2% - Merchant Benefit)

**Used For**: MCC (Multi-Currency Conversion) - Merchant settling in non-INR currency

**Calculation**:
```go
markdownPercent := 2.0
markdownExchangeRate := exchangeRate - (exchangeRate * markdownPercent / 100)
```

**Example**:
```
Exchange Rate: 83.00 INR/USD
Markdown: 2%
Markdown Rate: 83.00 - (83.00 * 0.02) = 81.34 INR/USD

Payment: 1000 INR
Merchant receives: 1000 / 81.34 = 12.29 USD

(Better than standard rate: 1000 / 83 = 12.05 USD)
```

**Purpose**: Gives merchant favorable rate (2% benefit) to incentivize accepting foreign currency.

### Markup (8% - Revenue Generation)

**Used For**: DCC (Dynamic Currency Conversion) - International cards paying in native currency

**Calculation**:
```go
markupPercent := 8.0
markupExchangeRate := exchangeRate + (exchangeRate * markupPercent / 100)
```

**Example**:
```
Exchange Rate: 90.00 INR/EUR
Markup: 8%
Markup Rate: 90.00 + (90.00 * 0.08) = 97.20 INR/EUR

Customer charged: 100 EUR
Merchant receives: 100 EUR * 97.20 = 9720 INR

(Merchant gets 720 INR extra revenue from markup)
```

**Purpose**: Revenue generation while providing convenience to international customers.

### Order of Operations (Critical)

```go
// CORRECT ORDER:
1. Subtract fee from amount (if CFB)
2. Apply markdown/markup to amount
3. Apply denomination factor
4. Apply rounding (ceil)
5. Calculate fee separately (at original rate)

// Example:
amount -= fee                              // Step 1
markdownRate := rate - rate * 0.02        // Step 2
baseAmount := ceil(markdownRate * amount * denominationFactor)  // Steps 3-4
baseFee := ceil(rate * fee * denominationFactor)               // Step 5
```

## Denomination Factor

### Purpose

Converts between currency denominations (cents ↔ dollars, paise ↔ rupees).

### Calculation

```go
denominationFactor := float64(conversionDenomination) / float64(baseDenomination)
```

**Example**:
```
USD base: 1 USD = 100 cents (baseDenomination = 100)
INR conversion: 1 INR = 100 paise (conversionDenomination = 100)
Denomination Factor: 100 / 100 = 1.0

JPY base: 1 JPY = 1 yen (baseDenomination = 1)
USD conversion: 1 USD = 100 cents (conversionDenomination = 100)
Denomination Factor: 100 / 1 = 100.0
```

### Critical Pattern

```go
// ALWAYS include denomination factor in conversion
convertedAmount := math.Ceil(exchangeRate * amount * denominationFactor)

// ❌ WRONG - Missing denomination factor
convertedAmount := math.Ceil(exchangeRate * amount)
```

### Division by Zero Protection

```go
// MUST protect against zero denominationCode location: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:626-632`

```go
func calculateConvertedAmountWithDenominations(amount int64, exchangeRate float64,
    markupPercent float64, baseDenomination int64, conversionDenomination int64,
    conversionExponent int64) int64 {

    if baseDenomination == 0 || conversionDenomination == 0 {
        return 0  // Prevent division by zero
    }
    denominationFactor := float64(conversionDenomination) / float64(baseDenomination)
    // ... calculation
}
```

## Ceiling Rounding

### Rule

**ALWAYS use math.Ceil() for rounding converted amounts, never floor or round.**

### Why Ceiling

- Ensures merchant never loses money due to rounding down
- Prevents accumulation of rounding errors
- Standard practice in payment processing

### Pattern

```go
✅ CORRECT - Ceiling rounding
baseAmount := math.Ceil(markdownExchangeRate * amount * denominationFactor)

❌ WRONG - Floor rounding (merchant loses money)
baseAmount := math.Floor(markdownExchangeRate * amount * denominationFactor)

❌ WRONG - Standard rounding (unpredictable)
baseAmount := math.Round(markdownExchangeRate * amount * denominationFactor)
```

### Example

```
Amount: 10003 cents ($100.03)
Rate: 1.35 SGD/USD
Calculation: 100.03 * 1.35 = 135.0405 SGD

Ceiling: 135.05 SGD ✅ (merchant gets full value)
Floor: 135.04 SGD ❌ (merchant loses 0.01 SGD)
Round: 135.04 SGD ❌ (rounds down, merchant loses)
```

## Three-Decimal Currency Handling

### Special Currencies

- **KWD** (Kuwaiti Dinar) - 1 KWD = 1000 fils
- **OMR** (Omani Rial) - 1 OMR = 1000 baisa
- **BHD** (Bahraini Dinar) - 1 BHD = 1000 fils

### Rounding Rule

**Round to nearest 10 (card network requirement).**

Code location: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:776-782`

```go
// roundOffIfApplicable rounds up to nearest 10 due to network requirement
func roundOffIfApplicable(amount int64, currencyExponent int64) int64 {
    if currencyExponent == 3 {  // KWD, OMR, BHD (3-decimal currencies)
        return cast.ToInt64(math.Ceil(cast.ToFloat64(amount)*0.1) / 0.1)
    }
    return amount
}
```

### Why Nearest 10

Card networks don't support odd values in 3-decimal currencies. Must round:
- 1.234 KWD → 1.240 KWD ✅
- 1.237 KWD → 1.240 KWD ✅
- 1.235 KWD → 1.240 KWD ✅

### Example

```go
amount := int64(1237)  // 1.237 KWD
currencyExponent := int64(3)

rounded := roundOffIfApplicable(amount, currencyExponent)
// Result: 1240 (1.240 KWD)

// Breakdown:
// 1237 * 0.1 = 123.7
// ceil(123.7) = 124
// 124 / 0.1 = 1240
```

## Complete Example

### MCC CFB Payment Flow

```go
// Input
amount := int64(100000)        // 1000 INR in paise
fee := int64(200)              // 2 INR fee
exchangeRate := 83.0           // 83 INR/USD
markdownPercent := 2.0
baseDenomination := int64(100) // INR paise
conversionDenomination := int64(100)  // USD cents

// Step 1: Subtract fee (CFB)
amount -= fee  // 99800 paise

// Step 2: Calculate markdown rate
markdownExchangeRate := exchangeRate - (exchangeRate * markdownPercent / 100)
// 83 - (83 * 0.02) = 81.34 INR/USD

// Step 3: Denomination factor
denominationFactor := float64(conversionDenomination) / float64(baseDenomination)
// 100 / 100 = 1.0

// Step 4: Convert amount with ceiling
baseAmount := math.Ceil(float64(amount) / 100 * markdownExchangeRate * denominationFactor)
// ceil(99800 / 100 * 81.34 * 1.0) = ceil(8121.732) = 8122 cents = $81.22

// Step 5: Convert fee at original rate (no markdown)
baseFee := math.Ceil(float64(fee) / 100 * exchangeRate * denominationFactor)
// ceil(200 / 100 * 83.0 * 1.0) = ceil(166.0) = 166 cents = $1.66

// Total settlement
total := baseAmount + baseFee
// 8122 + 166 = 8288 cents = $82.88
```

## Test Cases

### Test 1: Denomination Factor Included

```go
func TestConversion_DenominationFactor(t *testing.T) {
    amount := int64(100000)  // 1000 INR
    rate := 83.0
    baseDenom := int64(100)
    convDenom := int64(100)

    // With denomination factor
    denomFactor := float64(convDenom) / float64(baseDenom)
    withDenom := math.Ceil(float64(amount) / 100 * rate * denomFactor)

    // Without denomination factor (wrong)
    withoutDenom := math.Ceil(float64(amount) / 100 * rate)

    // Both should be same when denoms are equal, but pattern must include it
    assert.Equal(t, withDenom, withoutDenom, "When denoms equal, results match")

    // Test with different denoms (JPY example)
    baseDenomJPY := int64(1)
    denomFactorJPY := float64(convDenom) / float64(baseDenomJPY)
    assert.NotEqual(t, 1.0, denomFactorJPY, "Different denoms need factor")
}
```

### Test 2: Ceiling vs Floor Difference

```go
func TestRounding_Ceiling(t *testing.T) {
    amount := int64(10003)  // $100.03
    rate := 1.35

    ceiling := math.Ceil(float64(amount) / 100 * rate)
    floor := math.Floor(float64(amount) / 100 * rate)

    // Ceiling should be >= floor
    assert.GreaterOrEqual(t, ceiling, floor)

    // Merchant should get ceiling amount
    assert.Equal(t, int64(136), ceiling, "Merchant gets full value")
}
```

### Test 3: Three-Decimal Currency Rounding

```go
func TestThreeDecimalCurrency_RoundToNearest10(t *testing.T) {
    testCases := []struct {
        input    int64
        expected int64
    }{
        {1234, 1240},  // 1.234 KWD → 1.240 KWD
        {1237, 1240},  // 1.237 KWD → 1.240 KWD
        {1241, 1250},  // 1.241 KWD → 1.250 KWD
        {1000, 1000},  // 1.000 KWD → 1.000 KWD (already multiple of 10)
    }

    for _, tc := range testCases {
        result := roundOffIfApplicable(tc.input, 3)
        assert.Equal(t, tc.expected, result)
    }
}
```

## Source References

- **Markdown calculation**: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:699-715`
- **Denomination factor**: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:626-632`
- **Three-decimal rounding**: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:776-782`
- **Default rates**: `/payments-cross-border/internal/forex_processing/common/constants.go`

## Related Patterns

- [CFB Fee Handling](#cfb-fee-handling-patterns) - Fee subtraction before markdown
- [Currency Validation](#currency-validation-patterns) - Currency consistency checks
- [Common Anti-Patterns](#common-anti-patterns) - What to avoid


---

# Lifecycle Transitions

## Payment Lifecycle Stages

Cross-border payments go through multiple stages, and fee currency changes at specific transitions.

### Stage 1: Authorization

**What Happens**:
- Payment created, not yet captured
- Forex charges calculated (preview or actual)
- Fee in **payment currency** (if CFB)
- Amount validated against order

**Currency State**:
```
payment.currency = USD (payment currency)
payment.amount = 10000 cents ($100)
payment.fee = 200 cents ($2) // In payment currency
payment.status = authorized
```

**Code Location**: `/api/app/Models/Payment/Processor/Authorize.php`

### Stage 2: Capture

**What Happens**:
- Payment finalized, funds transferred
- Fee converted from payment currency to base currency
- Transaction entity created with base currency amounts
- Merchant settlement amount locked in

**Currency State** (MCC CFB):
```
// Before capture
payment.fee = 200 cents (USD)

// After capture
payment.fee = 16600 paise (INR) // Converted!
txn.fee = 16600 paise (INR)
txn.base_amount = ... (INR)
```

**Code Location**: `/api/app/Models/Payment/Processor/Capture.php`

### Stage 3: Settlement

**What Happens**:
- Merchant receives funds in base currency
- All amounts in base currency (INR for Indian merchants)
- Fee deducted from settlement (if DFB) or charged separately (if CFB)

**Currency State**:
```
All values in INR (merchant base currency)
```

## Critical Transition: Auth → Capture

### Fee Currency Conversion (MCC CFB)

**The Rule**: Fee must be converted from payment currency to base currency during capture.

Code location: `/api/app/Models/Payment/Processor/Capture.php`

```php
// MCC CFB Payments which are in authorized state will have the fees in payment currency,
// so we are converting it into base currency(INR) and sending it to Merchant Dashboard.
// If its a captured payment, fees are already stored in Base currency(INR)

if ($payment->getCurrency() !== Currency::INR
    and $payment->isFeeBearerCustomer()
    and $this->isMCCAppliedPayment($payment)) {
    // set fee values from txn as it will have INR For Both DCC or MCC or LRS Payments
    $payment->setFee($txn->getFee());
}
```

### Why This Matters

1. **Auth Stage**: Customer sees fee in their currency (USD)
2. **Capture Stage**: Merchant dashboard shows fee in INR
3. **Settlement**: Merchant receives/pays fee in INR

If conversion not done at capture, merchant dashboard shows wrong currency.

### Example Flow

```
Auth Stage:
- Customer: "You'll pay $2 fee"
- payment.fee = 200 cents (USD)
- payment.currency = USD

Capture Stage:
- Exchange rate: 83 INR/USD
- Convert fee: 200 cents / 100 * 83 = 166 INR = 16600 paise
- payment.fee = 16600 paise (INR) // UPDATED
- txn.fee = 16600 paise (INR)

Settlement:
- Merchant sees: "Fee: ₹166" (correct)
- Without conversion: "Fee: $2" (wrong currency!)
```

## DCC vs MCC Transitions

### DCC (Dynamic Currency Conversion)

```
Auth:
- payment.currency = EUR (customer native)
- payment.amount = 10000 cents (€100)
- payment.fee = 200 cents (€2, if CFB)
- markup applied: 8%

Capture:
- Convert EUR → INR (merchant base)
- Fee converted: €2 → ₹180 (approx)
- txn stores everything in INR

Settlement:
- Merchant receives INR
```

### MCC (Multi-Currency Conversion)

```
Auth:
- payment.currency = USD
- payment.amount = 10000 cents ($100)
- payment.fee = 200 cents ($2, if CFB)
- markdown will apply: 2%

Capture:
- Convert USD → INR (merchant base)
- Fee converted: $2 → ₹166
- txn stores everything in INR

Settlement:
- Merchant receives INR
```

## Transaction Entity Storage

### Rule

**Transaction entity ALWAYS stores amounts in base currency (INR).**

```php
// txn entity structure
$txn->amount = <amount in INR>
$txn->fee = <fee in INR>
$txn->tax = <tax in INR>
$txn->currency = Currency::INR
```

### Why

- Settlement happens in INR
- Accounting systems expect INR
- Merchant dashboard shows INR
- Reports/reconciliation in INR

### Payment vs Transaction

```
Payment Entity (Auth stage):
- currency: USD
- amount: 10000 (cents)
- fee: 200 (cents)

Transaction Entity (Capture stage):
- currency: INR
- amount: 830000 (paise)
- fee: 16600 (paise)
```

## Base Amount Recalculation

### When It Happens

Base amount recalculated when:
1. Fee provided in update flow (initial preview had no fee)
2. Exchange rate updated (rate changed between preview and actual)
3. Forex charges refreshed (expired cache)

### Code Pattern

```go
if !utils.IsEmpty(req.Fee) {
    // Re-calculate base amount with fee
    baseAmount, baseFee := calculateBaseAmountAndFee(
        amount, req.Fee, baseCurrency,
        forexChargesEntity.GetBaseCurrency(),
        forexChargesEntity.GetBaseForexRate(),
        forexChargesEntity.GetMarkdownPercent(),
    )
    forexChargesEntity.SetBaseAmount(baseAmount)
    forexChargesEntity.SetBaseFee(baseFee)
}
```

### Why Recalculation Needed

```
Preview (no fee yet):
- amount: 10000 USD
- fee: 0
- baseAmount: 10000 / 81.34 = 12294 paise

Actual Payment (fee known):
- amount: 10000 USD
- fee: 200 USD (2 USD)
- baseAmount: (10000 - 200) / 81.34 = 12050 paise // DIFFERENT!

If not recalculated:
- Merchant gets: 12294 + 166 = 12460 paise (WRONG - too much)
- Should get: 12050 + 166 = 12216 paise (CORRECT)
```

## Common Transition Errors

### Error 1: Fee Not Converted at Capture

```php
❌ WRONG
// Fee stays in payment currency
$payment->setFee(200);  // Still in USD!

✅ CORRECT
// Fee converted to base currency
if ($this->isMCCAppliedPayment($payment)) {
    $payment->setFee($txn->getFee());  // Now in INR
}
```

### Error 2: Base Amount Not Recalculated

```go
❌ WRONG
if req.Fee != nil {
    entity.SetFee(req.Fee)  // Only updating fee
    // baseAmount still calculated without fee!
}

✅ CORRECT
if !utils.IsEmpty(req.Fee) {
    baseAmount, baseFee := calculateBaseAmountAndFee(...)
    entity.SetBaseAmount(baseAmount)  // Recalculate!
    entity.SetBaseFee(baseFee)
}
```

### Error 3: Transaction Currency Mismatch

```php
❌ WRONG
$txn->currency = $payment->getCurrency();  // Might be USD!

✅ CORRECT
$txn->currency = Currency::INR;  // Always base currency
```

## Test Cases

### Test 1: Fee Conversion at Capture

```php
function testFeeConversionAtCapture_MCC_CFB() {
    $payment = new Payment([
        'currency' => 'USD',
        'amount' => 10000,
        'fee' => 200,  // In USD
        'is_fee_bearer_customer' => true,
    ]);

    // Before capture
    $this->assertEquals(200, $payment->getFee());

    // Capture
    $this->processor->capture($payment);

    // After capture - fee should be in INR
    $this->assertGreater($payment->getFee(), 200, 'Fee converted to INR');
    $this->assertEquals('INR', $payment->getTransaction()->getCurrency());
}
```

### Test 2: Base Amount Recalculation on Fee Update

```go
func TestBaseAmountRecalculation_OnFeeUpdate(t *testing.T) {
    // Initial: no fee
    entity := createForexCharges(10000, "USD", "INR")
    initialBase := entity.GetBaseAmount()

    // Update with fee
    req := &UpdateRequest{Fee: 200}
    UpdateForexCharges(entity, req)
    updatedBase := entity.GetBaseAmount()

    // Base amount should be less (fee subtracted)
    assert.Less(t, updatedBase, initialBase, "Base reduced after fee subtracted")
}
```

### Test 3: Transaction Entity Currency

```php
function testTransactionCurrency_AlwaysBaseCurrency() {
    $payment = new Payment(['currency' => 'USD', 'amount' => 10000]);

    $this->processor->capture($payment);

    $txn = $payment->getTransaction();
    $this->assertEquals('INR', $txn->getCurrency(), 'Txn always in base currency');
}
```

## Source References

- **API capture**: `/api/app/Models/Payment/Processor/Capture.php`
- **API authorize**: `/api/app/Models/Payment/Processor/Authorize.php`
- **Forex update**: `/payments-cross-border/internal/forex_processing/forex_charges/core.go:178-184`

## Related Patterns

- [CFB Fee Handling](#cfb-fee-handling-patterns) - Fee recalculation logic
- [Currency Validation](#currency-validation-patterns) - Currency at each stage
- [Common Anti-Patterns](#common-anti-patterns) - Transition mistakes


---

# Payments-Card Cross-Border Patterns

This document explains cross-border specific patterns in the payments-card service, focusing on DCC (Dynamic Currency Conversion), fee handling for CFB international payments, and EEFC settlement patterns.

## Overview

**Service**: payments-card (Razorpay Card Payment Service)
**Location**: `/Users/shrestha.k/rzp/payments-card`
**Cross-Border Package**: `pkg/cross_border/`

**What payments-card Handles**:
- DCC (Dynamic Currency Conversion) for international card payments
- CFB (Customer Fee Bearer) with international payments
- EEFC (Export Earners' Foreign Currency) settlement
- Gateway amount/currency preferences

**What payments-card Does NOT Handle**:
- Forex rate calculations (delegated to payments-cross-border via SDK)
- MCC (Multi-Currency Conversion) for merchants (handled by payments-cross-border)
- LRS payments (handled by payments-cross-border)

---

## Payment Entity Fields

**Location**: `internal/entities/payment/model.go`

### Base Currency Fields

```go
type Payment struct {
    Amount       int64  `json:"amount"`         // Payment amount in payment currency
    Currency     string `json:"currency"`        // Payment currency (INR, USD, etc.)
    BaseAmount   int64  `json:"base_amount"`    // Converted amount in base currency
    BaseCurrency string `json:"base_currency"`  // Merchant's base currency
    ForexApplied bool   `json:"forex_applied"`  // Flag indicating forex was applied

    // DCC fields
    PaymentMeta PaymentMeta `gorm:"references:PaymentId"` // Contains gateway_amount, gateway_currency
}
```

### Payment Meta Fields

**Location**: `internal/entities/payment_meta/model.go`

```go
type PaymentMeta struct {
    PaymentID         string `json:"payment_id"`
    GatewayAmount     int    `json:"gateway_amount"`     // DCC amount in cardholder currency
    GatewayCurrency   string `json:"gateway_currency"`   // DCC currency (USD, EUR, etc.)
    ForexRate         float64 `json:"forex_rate"`        // Exchange rate used
    DccMarkUpPercent  float64 `json:"dcc_mark_up_percent"` // 8% DCC markup
    MccForexRate      float64 `json:"mcc_forex_rate"`     // MCC exchange rate
    MccMarkdownPercent float64 `json:"mcc_markdown_percent"` // 2% MCC markdown
    DccOffered        bool   `json:"dcc_offered"`        // Whether DCC was offered
}
```

**Currency State**:
- **Before DCC**: `amount` and `currency` are merchant's (e.g., ₹1000 INR)
- **After DCC**: `gateway_amount` and `gateway_currency` added (e.g., $12.96 USD)
- **Base amounts**: `base_amount` and `base_currency` for settlement

---

## Critical Pattern 1: Gateway Amount/Currency Preference

**Problem**: When DCC is applied, which amount and currency should be used?

### Helper Functions

**Location**: `pkg/cross_border/helper.go:193-205`

```go
func getAmount(payment common.IPayment) float64 {
    if utils.IsEmpty(payment.GetGatewayAmount()) {
        return float64(payment.GetAmount())
    }
    return float64(payment.GetGatewayAmount())  // ✅ Prefer gateway amount for DCC
}

func getCurrency(payment common.IPayment) string {
    if utils.IsEmpty(payment.GetGatewayCurrency()) {
        return payment.GetCurrency()
    }
    return payment.GetGatewayCurrency()  // ✅ Prefer gateway currency for DCC
}
```

**When to Use**:
- For risk evaluation (`EvaluateRisk` API)
- For barricade (fraud) requests
- For external service calls that need actual charged amount

**Why This Matters**:
- DCC payments charge customer in gateway currency (USD, EUR)
- Using wrong amount/currency sends incorrect data to risk/fraud systems
- Validation logic must compare correct amounts

**Example**:

```go
// ❌ WRONG - Using payment.Amount for DCC payment
amount := payment.GetAmount()  // ₹1000 INR
// But customer was actually charged $12.96 USD

// ✅ CORRECT - Using gateway amount if DCC
amount := getAmount(payment)  // $12.96 USD (gateway_amount)
```

---

## Critical Pattern 2: Fee Handling for CFB International

**Problem**: When should fee be sent to forex charge creation?

### Fee Decision Logic

**Location**: `pkg/cross_border/helper.go:185-191`

```go
func shouldSendFeeForFXConversion(payment common.IPayment, req *request.ValidateRequest) bool {
    if !utils.IsEmpty(req) {
        return req.GetMerchant().GetFeeBearer() == constants.FeeBearerCustomer &&
               funk.ContainsString(req.GetMerchant().GetFeatures(), constants.AllowCFBInternational) &&
               req.Input.GetFee() > 0
    }
    return payment.GetFeeBearer() == constants.FeeBearerCustomer && payment.GetFees() > 0
}
```

**Conditions** (ALL must be true):
1. ✅ Fee bearer is **customer** (not platform/merchant)
2. ✅ Merchant has **AllowCFBInternational** feature enabled
3. ✅ Fee amount > 0

**Why This Matters**:
- Payments-cross-border needs fee for CFB calculation (subtract before markdown)
- Sending fee when not CFB causes incorrect forex calculations
- Missing fee for CFB causes merchant to receive wrong amount

**Usage Examples**:

**Location**: `pkg/cross_border/cross_border.go:124-126`

```go
// GetConvertedBaseAmount
pxbRequest := &pxbclient.ForexChargesInternalRequest{
    Amount:      req.Input.GetAmount(),
    Currency:    req.Input.GetCurrency(),
    MerchantID:  req.GetMerchant().GetID(),
}

if shouldSendFeeForFXConversion(nil, req) {
    pxbRequest.Fee = req.Input.GetFee()  // ✅ Only for CFB international
}
```

**Location**: `pkg/cross_border/cross_border.go:200-202`

```go
// CreateForexChargeInternal
pxbRequest := &pxbclient.ForexChargesInternalRequest{
    Amount:    payment.GetAmount(),
    Currency:  payment.GetCurrency(),
}

if shouldSendFeeForFXConversion(payment, nil) {
    pxbRequest.Fee = payment.GetFees()  // ✅ Only for CFB international
}
```

**Anti-Pattern**:

```go
// ❌ WRONG - Always sending fee
pxbRequest := &pxbclient.ForexChargesInternalRequest{
    Amount:   payment.GetAmount(),
    Currency: payment.GetCurrency(),
    Fee:      payment.GetFees(),  // ❌ Sent even for non-CFB!
}

// ❌ WRONG - Checking only fee bearer
if payment.GetFeeBearer() == "customer" {
    pxbRequest.Fee = payment.GetFees()  // ❌ Missing AllowCFBInternational check!
}
```

---

## Critical Pattern 3: DCC Blacklist Check

**Problem**: Some card IINs are blacklisted by networks for DCC.

### Blacklist Detection

**Location**: `internal/payment/processor/international.go:111-113`

```go
// Few IINs are marked as Blacklisted by networks for DCC, info related to dcc_blacklisted is
// present in the flows field of IIN details
if funk.Contains(iinDetails.GetFlows(), constants.DCC_BLACKLISTED) {
    // Skip DCC for this card
    return nil  // or handle appropriately
}
```

**Where IIN Details Come From**:
- IIN Service provides card metadata
- `flows` field contains array of strings
- Example: `["headless_otp", "dcc_blacklisted", "native_otp"]`

**Why This Matters**:
- Offering DCC to blacklisted cards causes network rejection
- Customer sees confusing error
- Must check BEFORE calling payments-cross-border for DCC rates

**Proper Flow**:

```go
// 1. Get IIN details
iinDetails := iinService.GetIINDetails(card.GetIin())

// 2. Check DCC blacklist
if funk.Contains(iinDetails.GetFlows(), constants.DCC_BLACKLISTED) {
    // Skip DCC entirely
    return processWithoutDCC(payment)
}

// 3. Proceed with DCC if not blacklisted
dccResponse := crossBorder.ForexChargesInternalForDCC(ctx, payment, merchant, input)
```

**Anti-Pattern**:

```go
// ❌ WRONG - Offering DCC without checking blacklist
dccResponse := crossBorder.ForexChargesInternalForDCC(ctx, payment, merchant, input)
// Card might be blacklisted → network rejection!
```

---

## Critical Pattern 4: Forex Applied Flag

**Problem**: Payment goes through multiple stages. How to know if forex was applied?

### Flag Usage

**Location**: `internal/entities/payment/model.go:111`

```go
type Payment struct {
    ForexApplied bool `gorm:"type:boolean;column:forex_applied" sql:"DEFAULT:false" json:"forex_applied"`
}
```

**Location**: `internal/payment/processor/international.go:189`

```go
payment.SetForexApplied(true)  // DCC is being applied
```

**When to Set**:
- ✅ When DCC is successfully applied (customer chooses native currency)
- ✅ When MCC is applied (multi-currency settlement)
- ❌ NOT for domestic INR-INR payments

**Why This Matters**:
- Gateway amount/currency getters check this flag
- Validation logic uses this to determine which amount to compare
- Analytics and reporting need to know if forex was applied

**Getter Implementation**:

**Location**: `internal/payment/processor/validation.go:1573` (comment)

```go
// For international payments, forex_applied will be set later when DCC/MCC is applied
```

**Location**: `internal/entities/payment/model.go` (custom getters)

```go
func (p *Payment) GetGatewayAmount() int64 {
    if p.ForexApplied && p.PaymentMeta.GatewayAmount > 0 {
        return int64(p.PaymentMeta.GatewayAmount)
    }
    return p.Amount
}

func (p *Payment) GetGatewayCurrency() string {
    if p.ForexApplied && p.PaymentMeta.GatewayCurrency != "" {
        return p.PaymentMeta.GatewayCurrency
    }
    return p.Currency
}
```

---

## Critical Pattern 5: EEFC Settlement

**Problem**: Export Earner's Foreign Currency (EEFC) account settlement has special forex handling.

### EEFC Update

**Location**: `pkg/cross_border/cross_border.go:222-243`

```go
// UpdateForexChargesForEEFC updates forex charges for EEFC settlement payments
// Extracts base_currency and base_amount from the payment object
// Uses default values: base_forex_rate=1, mark_down_percent=0
func (c *CrossBorder) UpdateForexChargesForEEFC(ctx context.Context, payment common.IPayment) (*pxbclient.UpdateForexChargesResponse, errors.IError) {
    baseCurrency := payment.GetBaseCurrency()
    baseAmount := payment.GetBaseAmount()

    entityType := constants.Payment
    entityID := payment.GetPaymentId()
    req := &pxbclient.UpdateForexChargesRequest{
        EntityType:   &entityType,
        EntityID:     &entityID,
        BaseCurrency: &baseCurrency,
        BaseAmount:   &baseAmount,
    }

    res, err := c.crossBorderSDKClient.PXBClient.UpdateForexCharges(ctx, req)
    if err != nil {
        return nil, err
    }
    return res, nil
}
```

**EEFC Specifics**:
1. **base_currency**: From payment entity (merchant's settlement currency)
2. **base_amount**: From payment entity (converted amount)
3. **base_forex_rate**: Defaults to 1 (no markup/markdown)
4. **markdown_percent**: Defaults to 0

**Why EEFC is Different**:
- EEFC accounts are special export earner accounts
- No markup/markdown applied (1:1 conversion)
- Settlement happens in foreign currency directly
- Used for merchants with overseas earnings

**When to Use**:
- Payment settled to EEFC account (check merchant configuration)
- After payment is authorized and before capture
- Only for international payments with EEFC enabled merchants

---

## Critical Pattern 6: Base Currency vs Payment Currency

**Problem**: Payment has multiple currency fields. Which one to use when?

### Currency Fields Explained

```go
type Payment struct {
    Currency     string  // Currency customer pays in (e.g., INR)
    BaseCurrency string  // Currency merchant settles in (e.g., USD for US merchant)

    // From PaymentMeta (DCC):
    GatewayCurrency string  // Currency offered to cardholder (e.g., EUR for European card)
}
```

### Usage Matrix

| Scenario | Currency | BaseCurrency | GatewayCurrency | ForexApplied |
|----------|----------|--------------|-----------------|--------------|
| **Domestic INR** | INR | INR | (empty) | false |
| **DCC (Indian merchant)** | INR | INR | USD | true |
| **MCC (US merchant)** | USD | USD | (empty) | true |
| **EEFC** | USD | USD | (empty) | true |

### Helper for Base Amount Conversion

**Location**: `pkg/cross_border/cross_border.go:109-164`

```go
func (c *CrossBorder) GetConvertedBaseAmount(ctx context.Context, req *request.ValidateRequest,
                                             isInternational bool, iinDetails common.IINModel) (int64, int64, errors.IError) {
    pxbRequest := &pxbclient.ForexChargesInternalRequest{
        Amount:          req.Input.GetAmount(),
        Currency:        req.Input.GetCurrency(),
        MerchantID:      req.GetMerchant().GetID(),
        MerchantCountry: req.GetMerchant().GetCountryCode(),
    }

    if shouldSendFeeForFXConversion(nil, req) {
        pxbRequest.Fee = req.Input.GetFee()
    }

    if !isInternational {
        // Domestic payment - conversion to merchant currency
        pxbRequest.EntityID = req.PaymentId
        pxbRequest.EntityType = constants.Payment
        pxbRequest.ConversionCurrency = req.GetMerchant().Currency
        // ... filter setup
    }

    res, err := c.crossBorderSDKClient.PXBClient.ForexChargesInternal(ctx, pxbRequest)
    return res.BaseAmount, res.BaseFee, nil  // Both in base currency
}
```

**Returns**:
- `BaseAmount`: Amount in merchant's base currency
- `BaseFee`: Fee in merchant's base currency

---

## Critical Pattern 7: DCC Flow Integration

**Problem**: How to properly integrate DCC in payment creation flow?

### DCC Creation Flow

**Location**: `pkg/cross_border/cross_border.go:264-291`

```go
func (c *CrossBorder) ForexChargesInternalForDCC(ctx context.Context, payment common.IPayment,
                                                  merchant *request.Merchant, input *request.Input) (*pxbclient.ForexChargesInternalResponse, errors.IError) {
    card, _ := payment.GetCard()
    pxbRequest := &pxbclient.ForexChargesInternalRequest{
        Amount:                payment.GetAmount(),
        Currency:              payment.GetCurrency(),
        ConversionCurrency:    input.GetDCCCurrency(),        // Customer's choice
        CurrencyRequestID:     input.GetCurrencyRequestId(),  // For rate locking
        EntityType:            constants.Payment,
        EntityID:              payment.GetPaymentId(),
        MerchantID:            merchant.GetID(),
        SavePreferredCurrency: input.IsSavePreferredCurrency(),  // Save customer preference
        CardFingerprint:       card.GetGlobalFingerprint(),       // For preference lookup
        Filters: map[string]string{
            "method": "card",
        },
    }

    if shouldSendFeeForFXConversion(payment, nil) {
        pxbRequest.Fee = payment.GetFees()
    }

    res, err := c.crossBorderSDKClient.PXBClient.ForexChargesInternal(ctx, pxbRequest)
    return res, err
}
```

**DCC Fields Explained**:
1. **ConversionCurrency**: Customer's chosen currency (from `dcc_currency` in request)
2. **CurrencyRequestID**: Locks exchange rate from DCC info API
3. **SavePreferredCurrency**: Save customer's currency choice for future payments
4. **CardFingerprint**: Used to retrieve saved preference

### DCC Info Flow

**Location**: `pkg/cross_border/cross_border.go:293-319`

```go
func (c *CrossBorder) ForexChargesInternalForDCCInfo(ctx context.Context, payment common.IPayment,
                                                      dccInfoParams *request.DccInfoParams) (*pxbclient.ForexChargesInternalResponse, errors.IError) {
    pxbRequest := &pxbclient.ForexChargesInternalRequest{
        Amount:                payment.GetAmount(),
        Currency:              payment.GetCurrency(),
        MerchantID:            payment.GetMerchantId(),
        AllCurrencies:         true,  // ✅ Return all DCC currency options
        Filters: map[string]string{
            "method": "card",
        },
    }

    // Extract provider from notes if present
    if ok, provider := utils.ExtractProviderFromNotes(payment.GetNotes()); ok {
        pxbRequest.Filters["provider"] = provider
    }

    if shouldSendFeeForFXConversion(payment, nil) {
        pxbRequest.Fee = payment.GetFees()
    }

    res, err := c.crossBorderSDKClient.PXBClient.ForexChargesInternal(ctx, pxbRequest)
    return res, err
}
```

**DCC Info Response**:
- Returns multiple currency options (USD, EUR, GBP, etc.)
- Each with exchange rate and markup percentage
- Customer chooses from these options
- `currency_request_id` locks the rate for 15 minutes

---

## Detection Patterns for Code Review

### Pattern 1: Missing CFB International Check

**🚨 CRITICAL**

```go
// ❌ WRONG - Fee sent without checking CFB international
pxbRequest.Fee = payment.GetFees()

// ✅ CORRECT
if shouldSendFeeForFXConversion(payment, req) {
    pxbRequest.Fee = payment.GetFees()
}
```

**Where to Check**:
- Any call to `ForexChargesInternal`
- Any call to `CreateForexChargeInternal`
- Any call to `ForexChargesInternalForDCC`

---

### Pattern 2: Wrong Amount/Currency for DCC

**🚨 CRITICAL**

```go
// ❌ WRONG - Using payment.Amount for DCC payment
riskRequest.Amount = payment.GetAmount()
riskRequest.Currency = payment.GetCurrency()

// ✅ CORRECT
riskRequest.Amount = getAmount(payment)      // Prefers gateway_amount
riskRequest.Currency = getCurrency(payment)  // Prefers gateway_currency
```

**Where to Check**:
- Risk/fraud API calls
- External service integrations
- Validation logic comparing amounts

---

### Pattern 3: Missing DCC Blacklist Check

**⚠️ HIGH-PRIORITY**

```go
// ❌ WRONG - Offering DCC without blacklist check
dccResponse := ForexChargesInternalForDCC(...)

// ✅ CORRECT
if funk.Contains(iinDetails.GetFlows(), constants.DCC_BLACKLISTED) {
    // Skip DCC
    return processWithoutDCC(payment)
}
dccResponse := ForexChargesInternalForDCC(...)
```

**Where to Check**:
- Before any DCC info API call
- Before any DCC forex charge creation

---

### Pattern 4: Missing Forex Applied Flag

**⚠️ HIGH-PRIORITY**

```go
// ❌ WRONG - Not setting forex_applied after DCC
dccResponse := ForexChargesInternalForDCC(...)
payment.SetGatewayAmount(dccResponse.GatewayAmount)
payment.SetGatewayCurrency(dccResponse.GatewayCurrency)

// ✅ CORRECT
dccResponse := ForexChargesInternalForDCC(...)
payment.SetGatewayAmount(dccResponse.GatewayAmount)
payment.SetGatewayCurrency(dccResponse.GatewayCurrency)
payment.SetForexApplied(true)  // ✅ Set flag
```

**Where to Check**:
- After successful DCC application
- After successful MCC application

---

### Pattern 5: EEFC Wrong Parameters

**⚠️ HIGH-PRIORITY**

```go
// ❌ WRONG - Hardcoding values for EEFC
req := &pxbclient.UpdateForexChargesRequest{
    BaseCurrency: "USD",  // ❌ Should come from payment
    BaseAmount:   10000,  // ❌ Should come from payment
}

// ✅ CORRECT
req := &pxbclient.UpdateForexChargesRequest{
    BaseCurrency: &payment.GetBaseCurrency(),  // ✅ From payment
    BaseAmount:   &payment.GetBaseAmount(),    // ✅ From payment
}
```

---

## Testing Checklist

### For CFB International Payments

- [ ] Fee sent only when `shouldSendFeeForFXConversion()` returns true
- [ ] Fee NOT sent for non-CFB payments
- [ ] Fee NOT sent when merchant doesn't have AllowCFBInternational
- [ ] Test with fee = 0 (should not send fee)

### For DCC Payments

- [ ] IIN blacklist checked before offering DCC
- [ ] Gateway amount/currency used for risk/fraud calls
- [ ] `forex_applied` flag set after DCC application
- [ ] Currency request ID passed for rate locking
- [ ] Card fingerprint passed for preference saving

### For EEFC Settlement

- [ ] Base currency from payment entity (not hardcoded)
- [ ] Base amount from payment entity (not recalculated)
- [ ] Only called for EEFC-enabled merchants
- [ ] Called before capture (not after)

### For General Cross-Border

- [ ] Correct currency used based on context (payment vs gateway vs base)
- [ ] All forex API calls have proper error handling
- [ ] Payment meta properly populated after forex application

---

## Related Documentation

- [MCC/DCC Lifecycle Flows](#mccdcc-lifecycle-flows-and-currency-states) - Currency states at different stages
- [CFB Fee Handling](#cfb-fee-handling-patterns) - General CFB patterns
- [Currency Validation](#currency-validation-patterns) - Currency consistency rules

---

## Code Locations Reference

### Cross-Border SDK Integration
- **Initialization**: `pkg/cross_border/cross_border.go:45-94`
- **Base Amount Conversion**: `pkg/cross_border/cross_border.go:109-164`
- **DCC Creation**: `pkg/cross_border/cross_border.go:264-291`
- **DCC Info**: `pkg/cross_border/cross_border.go:293-319`
- **EEFC Update**: `pkg/cross_border/cross_border.go:222-243`

### Helper Functions
- **Should Send Fee**: `pkg/cross_border/helper.go:185-191`
- **Get Amount**: `pkg/cross_border/helper.go:193-198`
- **Get Currency**: `pkg/cross_border/helper.go:200-205`

### Payment Entity
- **Model**: `internal/entities/payment/model.go:37-115`
- **Payment Meta**: `internal/entities/payment_meta/model.go`

### DCC Processing
- **Blacklist Check**: `internal/payment/processor/international.go:111-113`
- **Forex Applied**: `internal/payment/processor/international.go:189`
- **Checkout Activity**: `internal/payment/processor/checkout_activity.go:123-124`

---

**Version**: 1.0.0
**Last Updated**: 2026-02-19
**Maintainer**: Cross-Border Code Review Skill


---

# MCC/DCC Lifecycle Flows and Currency States

This document provides detailed flow-specific information about cross-border payment types (MCC, DCC, LRS) and explains **exactly which currency** the fee and amount are in at each lifecycle stage. This is critical for preventing currency mismatch bugs in pg-router and payments-cross-border.

## Understanding the Problem

**The Core Issue**: In cross-border CFB (Customer Fee Bearer) payments, the **fee currency changes** during the payment lifecycle, but the **amount currency stays the same**. Mixing currencies in arithmetic operations causes incorrect calculations.

**Real Production Incident** (pg-router PR #3701):
```go
// PostProcessingCapture - WRONG
func adjustPaymentAmount(payment *Payment) int64 {
    adjustedAmount := payment.Amount     // USD (MCC currency)
    if feeBearer == "customer" {
        adjustedAmount -= payment.Fee    // INR (base currency after capture)
    }
    // Result: $100 - ₹166 = nonsense!
}
```

---

## Payment Type Definitions

### MCC (Multi-Currency Conversion)

**What**: Merchant settles in a currency different from INR with a 2% markdown benefit.

**Example Flow**:
- Customer pays: $100 USD
- Merchant (India) settles in: $100 USD (not converted to INR)
- Markdown benefit: 2% (merchant gets better exchange rate)
- Fee: $2 USD (at auth) → ₹166 INR (at capture, if CFB)

**Merchant Types**:
- India-based merchant with multi-currency settlement enabled
- Singapore merchant (settles in SGD)
- Malaysia merchant (settles in MYR)

**Entity**: Handled by `forex_charges` entity in payments-cross-border

---

### DCC (Dynamic Currency Conversion)

**What**: International cardholder pays in their native currency instead of merchant's currency.

**Example Flow**:
- Merchant charges: ₹1,000 INR
- US cardholder sees: $12.96 USD (with 8% markup)
- Customer chooses: Pay in USD
- Merchant settles: ₹1,000 INR
- Fee: ₹20 INR (no currency conversion, merchant in India)

**Key Difference from MCC**: In DCC, merchant **always** settles in their base currency (INR for India). Only the cardholder's view changes.

**Entity**: Handled by `forex_charges` entity with gateway_amount/gateway_currency fields

---

### LRS (Liberalized Remittance Scheme)

**What**: Outward remittances from India for permitted purposes (education, medical, gifts).

**Example Flow**:
- User sends: $10,000 USD
- Debited from INR account: ₹8,30,000 INR (at current rate)
- Fee: ₹500 INR (always in INR)
- Beneficiary receives: $10,000 USD

**Key Point**: LRS always has fee in INR since source is India.

**Entity**: Handled by `forex_details` entity (simpler than forex_charges)

---

## Lifecycle Stages and Currency States

### Stage 1: Authorization (Auth)

**When**: Payment is authorized by the gateway but not yet captured.

**Currency States**:

| Field | MCC Example | DCC Example | LRS Example |
|-------|-------------|-------------|-------------|
| `payment.Amount` | $100 USD | ₹1,000 INR | $10,000 USD |
| `payment.Currency` | USD | INR | USD |
| `payment.Fee` | **$2 USD** | **₹20 INR** | **₹500 INR** |
| `fee_bearer` | customer | platform | customer |
| `transaction.Amount` | $100 USD | ₹1,000 INR | $10,000 USD |
| `transaction.Fee` | **$2 USD** | **₹20 INR** | **₹500 INR** |

**Critical Point for MCC CFB**:
- At authorization, **fee is in payment currency** (USD)
- Formula: `netAmount = $100 - $2 = $98 USD` ✅ **CORRECT** (same currency)

**Code Location (pg-router)**:
- `internal/payments/core/create.go` - Payment creation
- Fee calculated by Scrooge service in payment currency

**Code Location (payments-cross-border)**:
- `internal/forex_processing/forex_charges/core.go:236` - Fee subtracted before markdown
- `amount -= fee` (both in payment currency at this stage)

---

### Stage 2: Capture (Critical Transition)

**When**: Merchant captures the authorized payment to trigger settlement.

**What Changes**: For **cross-border CFB payments**, fee is **converted to base currency**.

**Currency States AFTER Capture**:

| Field | MCC Example | DCC Example | LRS Example |
|-------|-------------|-------------|-------------|
| `payment.Amount` | **$100 USD** (unchanged) | **₹1,000 INR** (unchanged) | **$10,000 USD** (unchanged) |
| `payment.Currency` | USD | INR | USD |
| `payment.Fee` | **₹166 INR** (converted!) | **₹20 INR** (unchanged) | **₹500 INR** (unchanged) |
| `transaction.Currency` | **INR** (base currency) | **INR** | **INR** |
| `transaction.Fee` | **₹166 INR** | **₹20 INR** | **₹500 INR** |

**Critical Point for MCC CFB**:
- After capture, **fee is in base currency** (INR)
- After capture, **amount is still in payment currency** (USD)
- Formula: `$100 - ₹166` ❌ **CURRENCY MISMATCH!**

**Why Fee Currency Changes**:
1. Transaction entity stores everything in merchant's base currency (INR for India)
2. Fee needs to be in same currency as transaction for ledger entries
3. Conversion happens at capture: `fee_INR = fee_USD * exchange_rate`

**Code Location (payments-cross-border)**:
- Fee conversion at capture (for CFB + MCC):
  ```go
  // Line 241 in forex_charges/core.go
  baseFee := cast.ToInt64(math.Ceil(exchangeRate * cast.ToFloat64(fee) *
                                   denominationFactor))
  ```

**Code Location (pg-router)**:
- Capture processing: `internal/payments/core/capture.go`
- Transaction entity created with fee in INR
- Payment entity updated with converted fee

---

### Stage 3: Post-Processing Capture (Where PR #3701 Issue Occurred)

**When**: After capture completes, during order status update and ledger processing.

**Currency States** (Same as Stage 2):

| Field | MCC CFB Example | MCC Non-CFB | Domestic INR |
|-------|-----------------|-------------|--------------|
| `payment.Amount` | **$100 USD** | **$100 USD** | **₹1,000 INR** |
| `payment.Currency` | USD | USD | INR |
| `payment.Fee` | **₹166 INR** ⚠️ | **₹166 INR** | **₹20 INR** |
| Expected `order.amount_paid` | **$98 USD worth in paise** | **$100 USD** | **₹980 INR** |

**The Bug in PR #3701**:
```go
// internal/payments/core/capture.go:637-652
func (c *Core) adjustPaymentAmount(payment *cps.Payment) int64 {
    adjustedAmount := payment.Amount  // $100 USD = 10000 cents

    if feeBearer == merchant_constants.FeeBearerCustomer {
        adjustedAmount -= payment.Fee  // ₹166 INR = 16600 paise
    }
    // Result: 10000 - 16600 = -6600 ❌ WRONG!
    // Actual issue: Mixing USD cents with INR paise
}
```

**Why This Happens**:
1. `PostProcessingCapture` runs **after** capture completes
2. At this stage, `payment.Fee` has already been converted to INR (base currency)
3. But `payment.Amount` is still in USD (payment currency)
4. Direct subtraction produces incorrect order paid amount

**Correct Approach**:
```go
// Option 1: Convert fee back to payment currency
feeInPaymentCurrency := convertCurrency(payment.Fee, "INR", payment.Currency)
adjustedAmount = payment.Amount - feeInPaymentCurrency

// Option 2: Use transaction entity (already in base currency)
adjustedAmount = transaction.Amount - transaction.Fee  // Both in INR

// Option 3: Validate currencies match before arithmetic
if payment.Currency != baseCurrency {
    // Fee is in base currency, amount in payment currency
    // Need conversion or use different source
}
```

---

## Service Boundaries and Context Differences

### payments-cross-border Service

**Scope**: Handles forex charge calculations and DCC offerings.

**Context**:
- Operates during **authorization** stage
- All calculations done **before** capture
- Fee and amount are **always in same currency** at calculation time
- Formula works correctly:
  ```go
  amount -= fee  // Both in payment currency
  baseAmount := ceil(markdownRate * amount * denominationFactor)
  baseFee := ceil(exchangeRate * fee * denominationFactor)
  ```

**Currency State**:
- Input: Amount and fee in payment currency
- Output: Base amount and base fee in merchant base currency
- Conversion happens together, no mismatch

---

### pg-router Service

**Scope**: Handles payment lifecycle, routing, and order management.

**Context**:
- Operates across **all lifecycle stages**
- Must handle currency changes between stages
- Payment entity persists through auth → capture → settlement
- Fee currency changes at capture, but amount currency doesn't

**Currency State Challenges**:
- Authorization: Fee in payment currency
- Post-capture: Fee in base currency, amount still in payment currency
- Must be aware of which stage the code is running in

**Critical Areas to Watch**:
1. **Post-capture processing**: Fee already converted, amount not
2. **Order amount calculations**: Must match currencies before arithmetic
3. **Validation logic**: Comparing payment amount to order amount

---

## Flow-Specific Checklist

### MCC CFB Payments

**Critical Checks**:
- [ ] **Auth stage**: Fee subtracted from amount (both in payment currency) ✅
- [ ] **Capture stage**: Fee converted to base currency
- [ ] **Post-capture**: Validate currencies before amount - fee
- [ ] **Order paid amount**: Use converted values or transaction entity
- [ ] **Settlement**: Use base currency values from transaction

**Test Scenarios**:
```go
// MCC CFB: USD payment, Indian merchant
payment := Payment{
    Amount: 10000,      // $100.00 USD in cents
    Currency: "USD",
    Fee: 16600,         // ₹166 INR in paise (after capture)
    FeeBearer: "customer",
}

// At auth stage
feeAtAuth := 200       // $2.00 USD
netAtAuth := 10000 - 200 = 9800  // $98.00 USD ✅

// At post-capture (WRONG if not careful)
netPostCapture := 10000 - 16600 = -6600  // ❌ Currency mismatch!

// At post-capture (CORRECT)
feeUSD := convertToUSD(16600, rate)  // ₹166 → $2.00
netPostCapture := 10000 - feeUSD = 9800  // ✅
```

---

### DCC Payments

**Critical Checks**:
- [ ] **Auth stage**: Gateway amount offered in cardholder currency
- [ ] **Capture stage**: Merchant settles in base currency (no fee conversion needed)
- [ ] **Payment meta**: DCC flags set correctly (dcc_offered, mcc_applied)

**Currency States**:
- Payment amount: Always in merchant currency (INR for India)
- Gateway amount: In cardholder's native currency (USD, EUR, GBP, etc.)
- Fee: Always in merchant currency (no conversion)

**No CFB Currency Issue**: Since merchant is in India, fee is already in INR. No conversion at capture.

---

### LRS Payments

**Critical Checks**:
- [ ] **Auth stage**: Amount in foreign currency, fee in INR
- [ ] **Capture stage**: No fee conversion (already in INR)
- [ ] **FIRS reporting**: Amounts reported correctly to RBI

**Currency States**:
- Payment amount: Foreign currency (USD, EUR, etc.)
- Fee: Always INR (source is India)
- Transaction: All in INR

**No CFB Currency Issue**: Fee is already in INR, so no conversion happens.

---

## Currency Mismatch Detection Patterns

### Pattern 1: Direct Arithmetic Without Currency Check

**Wrong**:
```go
if payment.Currency != "INR" && feeBearer == "customer" {
    adjustedAmount := payment.Amount - payment.Fee  // ❌ Mismatch!
}
```

**Why Wrong**: After capture, `payment.Fee` is in INR but `payment.Amount` is in MCC currency (e.g., USD).

**Correct**:
```go
if payment.Currency != baseCurrency && feeBearer == "customer" {
    // Option 1: Validate currencies match
    if payment.FeeCurrency != payment.Currency {
        logger.Error("Fee currency mismatch detected")
        // Convert or use transaction entity
    }

    // Option 2: Use transaction entity (already in base currency)
    adjustedAmount := transaction.Amount - transaction.Fee

    // Option 3: Convert fee to payment currency
    feeConverted := convertCurrency(payment.Fee, baseCurrency, payment.Currency)
    adjustedAmount := payment.Amount - feeConverted
}
```

---

### Pattern 2: Stage-Aware Processing

**Stage Detection**:
```go
func isPreCapture(payment *Payment) bool {
    return payment.Status == "authorized" || payment.Status == "created"
}

func isPostCapture(payment *Payment) bool {
    return payment.Status == "captured" || payment.CapturedAt > 0
}

func adjustPaymentAmount(payment *Payment) int64 {
    adjustedAmount := payment.Amount

    if feeBearer == "customer" {
        if isPreCapture(payment) {
            // Fee in payment currency
            adjustedAmount -= payment.Fee  // ✅ Safe
        } else if isPostCapture(payment) {
            // Fee converted to base currency
            if payment.Currency == baseCurrency {
                adjustedAmount -= payment.Fee  // ✅ Safe (same currency)
            } else {
                // MCC case: fee in INR, amount in USD
                feeConverted := convertToPaymentCurrency(payment.Fee)
                adjustedAmount -= feeConverted  // ✅ Safe
            }
        }
    }

    return adjustedAmount
}
```

---

### Pattern 3: Use Transaction Entity for Post-Capture

**Transaction Entity** (always in base currency):
```go
type Transaction struct {
    Amount   int64  // Base currency (INR)
    Fee      int64  // Base currency (INR)
    Currency string // Always base currency
}
```

**Correct Usage**:
```go
// Post-capture: Use transaction entity
func calculateOrderPaidAmount(payment *Payment, txn *Transaction) int64 {
    if payment.Currency == baseCurrency {
        // Domestic or already in base currency
        return payment.Amount - payment.Fee  // ✅ Safe
    } else {
        // Cross-border MCC: Use transaction entity
        return txn.Amount - txn.Fee  // ✅ Safe (both in INR)
    }
}
```

---

## Key Takeaways

### For Reviewers

1. **MCC CFB is the high-risk case**: Fee currency changes at capture, amount doesn't
2. **PostProcessingCapture is critical**: This is where currency mismatch happens
3. **Look for `payment.Amount - payment.Fee`**: Red flag in post-capture code
4. **Transaction entity is safe**: Always in base currency, no mismatch

### For Developers

1. **Know your lifecycle stage**: Pre-capture vs post-capture matters
2. **Validate currencies before arithmetic**: Add explicit checks
3. **Use transaction entity post-capture**: It has consistent currency
4. **Test MCC CFB scenarios**: This is where bugs hide

### Currency States Summary Table

| Stage | MCC CFB | Fee Currency | Amount Currency | Safe to Subtract? |
|-------|---------|--------------|-----------------|-------------------|
| **Auth** | Yes | Payment currency (USD) | Payment currency (USD) | ✅ YES |
| **Pre-capture** | Yes | Payment currency (USD) | Payment currency (USD) | ✅ YES |
| **Capture** | Yes | **Base currency (INR)** | Payment currency (USD) | ❌ NO |
| **Post-capture** | Yes | **Base currency (INR)** | Payment currency (USD) | ❌ NO |
| **Settlement** | Yes | Base currency (INR) | Base currency (INR)* | ✅ YES* |

*Use transaction entity for settlement

---

## Related Documentation

- [CFB Fee Handling](#cfb-fee-handling-patterns) - Fee subtraction patterns
- [Currency Validation](#currency-validation-patterns) - Currency consistency rules
- [Lifecycle Transitions](#lifecycle-transitions) - General lifecycle stages
- [Common Anti-Patterns](common-antipatterns.md#anti-pattern-3) - Currency mismatch examples

---

**Version**: 1.0.0
**Last Updated**: 2026-02-18
**Maintainer**: Cross-Border Code Review Skill


---

# Cross-Border Patterns in Scrooge (Refunds Service)

## Overview

**Scrooge** is Razorpay's collections and refunds management service. For cross-border payments, scrooge handles **refund processing** for DCC (Dynamic Currency Conversion), MCC (Multi-Currency Conversion), and LRS (Liberalized Remittance Scheme) payments.

**Service Responsibility**:
- Calculate refund amounts in gateway currency (DCC refunds)
- Handle denomination factors for 3-decimal currencies
- Apply forex rates and markup/markdown for partial refunds
- Ensure refund amounts don't exceed payment amounts (money leak prevention)

**Key Difference from Payment Processing**:
- Payments use `Math.Ceil()` for currency conversion (favor customer)
- Refunds use `Math.Floor()` for currency conversion (prevent money leaks)
- This asymmetry is **intentional** to protect against partial refund rounding errors

---

## Critical Pattern 1: DCC Gateway Amount Calculation (Math.Floor)

### The Pattern

**Location**: `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go:456-471`

```go
func GetGatewayAmountForDccPayment(ctx *gin.Context, paymentMeta map[string]interface{},
    input map[string]interface{}, payment map[string]interface{}) int64 {
    var gatewayAmount int64

    refundAmount, _ := utils.ConvertInterfaceToFloat(ctx, input[constants.ParamAmount], 64)
    forexRate, _ := utils.ConvertInterfaceToFloat(ctx, paymentMeta[refundUtils.ForexRate], 64)
    markUpPercent, _ := utils.ConvertInterfaceToFloat(ctx, paymentMeta[refundUtils.DccMarkUpPercent], 64)

    // Denomination factor for 3-decimal currencies (KWD, OMR, BHD)
    denominationFactor, _ := utils.GetGatewayCurrencyDenominationFactor(ctx, payment)
    convertedAmount := refundAmount * forexRate * denominationFactor

    // CRITICAL: Math.Floor (not Ceil!) to prevent money leaks
    gatewayAmount = int64(math.Floor(convertedAmount + (markUpPercent*convertedAmount)/100))
    return gatewayAmount
}
```

### Why Math.Floor?

**Problem with Math.Ceil in Refunds**:
```
Payment: ₹100 → $1.32 USD (with 8% markup, rate 83.00)
- Calculation: Ceil((100 * 0.012048) * 1.08) = Ceil(1.301184) = $1.32 ✅

Partial Refund 1: ₹50 → $0.66 USD
- With Ceil: Ceil((50 * 0.012048) * 1.08) = Ceil(0.650592) = $0.66

Partial Refund 2: ₹50 → $0.66 USD
- With Ceil: Ceil((50 * 0.012048) * 1.08) = Ceil(0.650592) = $0.66

Total Refunds: $0.66 + $0.66 = $1.32
Payment Amount: $1.32

Problem: Both match! BUT...

If we do THREE partial refunds of ₹33.33 each:
- Refund 1: Ceil(0.433728) = $0.44
- Refund 2: Ceil(0.433728) = $0.44
- Refund 3: Ceil(0.433728) = $0.44
Total: $1.32

But what if we refund ₹25 four times?
- Each: Ceil(0.325296) = $0.33
- Total: $1.32

HOWEVER, if we refund ₹20 five times:
- Each: Ceil(0.260237) = $0.27
- Total: $1.35 ❌ MONEY LEAK! Refunded more than payment!
```

**Solution with Math.Floor**:
```
Partial Refund: ₹20 → $0.26 USD (Floor, not Ceil)
- Calculation: Floor((20 * 0.012048) * 1.08) = Floor(0.260237) = $0.26

5 refunds: $0.26 * 5 = $1.30 < $1.32 ✅ Safe!
```

**Trade-off**: Customer may lose up to $0.01 per refund, but merchant is protected from money leaks.

### When This Applies

**Triggers**:
- DCC payments (payment_meta has `is_dcc = true`)
- Partial refunds on international payments
- Any refund calculation involving `forex_rate` and `dcc_mark_up_percent`

**Does NOT Apply**:
- Domestic INR refunds (no forex conversion)
- LRS refunds (uses pre-calculated gateway_amount from payment_meta)
- Full refunds (uses exact payment gateway amount)

---

## Critical Pattern 2: Three-Decimal Currency Rounding

### The Pattern

**Location**: `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go:483-494`

```go
// Round down gateway amount to nearest 0 for three decimal currencies
// due to network requirement.
// Since we cant send a non-zero digit in the last unit to networks
// neither we can round up, due to money leak concerns
// during partial refunds,
// here gateway amount is rounded down. End customers
// will bear loss upto 0.020 KWD/OMR/BHD, etc.
// Such is life !!!
func RoundDownGatewayAmountIfApplicable(
    ctx *gin.Context,
    paymentMeta map[string]interface{},
    input map[string]interface{},
    gatewayAmount int64) int64 {

    if refundUtils.IsLastDigitZeroCurrencies(
        utils.ConvertInterfaceToString(paymentMeta[refundUtils.GatewayCurrency])) == true {
        gatewayAmount = int64((gatewayAmount / 10) * 10)  // Round down to nearest 10
    }

    return gatewayAmount
}
```

**Three-Decimal Currencies**:
```go
// /scrooge/app/services/internals/refund/utils/currency.go
var ThreeDecimalCurrencies = []string{
    "BHD",  // Bahraini Dinar
    "KWD",  // Kuwaiti Dinar
    "OMR",  // Omani Rial
}
```

### Why Round Down?

**Network Requirement**: Card networks (Visa, Mastercard) reject transactions with non-zero last digit for 3-decimal currencies.

**Example**:
```
Refund Amount: 1.234 KWD = 1234 fils
After rounding: 1.230 KWD = 1230 fils (last digit forced to 0)

Calculation:
  gatewayAmount = 1234
  rounded = (1234 / 10) * 10 = 123 * 10 = 1230 ✅

Customer loses: 4 fils (~$0.013)
```

**Slack Context**:
- https://razorpay.slack.com/archives/C01LK94TC69/p1690199516379309
- https://razorpay.slack.com/archives/C3Y0UA0CB/p1690359032652129

### When This Applies

**Triggers**:
- Gateway currency is KWD, OMR, or BHD
- DCC refunds only (not LRS or domestic)
- Applied AFTER `GetGatewayAmountForDccPayment()` calculation

**Order of Operations**:
```go
// Line 322-323 in create_v2_prepare_refund_params.go
gatewayAmount = GetGatewayAmountForDccPayment(ctx, paymentMeta, input, payment)
gatewayAmount = RoundDownGatewayAmountIfApplicable(ctx, paymentMeta, input, gatewayAmount)
```

---

## Critical Pattern 3: Denomination Factor Usage

### The Pattern

**Location**: `/scrooge/app/utils/utils.go:1201-1216`

```go
func GetGatewayCurrencyDenominationFactor(
    ctx *gin.Context,
    payment map[string]interface{}) (float64, rzperrors.IError) {

    gatewayCurrencyDenomination, ierr := ConvertInterfaceToFloat(ctx,
        payment[constants.GatewayCurrencyDenomination], 64)
    if (ierr != nil) || (IsNumericZero(gatewayCurrencyDenomination) == true) {
        return 1, nil  // Default to 1 if missing
    }

    paymentCurrencyDenomination, ierr := ConvertInterfaceToFloat(ctx,
        payment[constants.PaymentRawCurrencyDenomination], 64)
    if (ierr != nil) || (IsNumericZero(paymentCurrencyDenomination) == true) {
        return 1, nil  // Default to 1 if missing
    }

    return float64(gatewayCurrencyDenomination / paymentCurrencyDenomination), nil
}
```

### Denomination Factor Explained

**Purpose**: Convert amounts between currencies with different decimal places.

**Examples**:

**Case 1: INR (2 decimals) → USD (2 decimals)**
```
payment_currency_denomination = 100 (paise)
gateway_currency_denomination = 100 (cents)
denomination_factor = 100 / 100 = 1.0

Refund: ₹100.00 = 10000 paise
Converted: 10000 * 0.012 * 1.0 = 120 cents = $1.20 ✅
```

**Case 2: INR (2 decimals) → JPY (0 decimals)**
```
payment_currency_denomination = 100 (paise)
gateway_currency_denomination = 1 (yen, no decimals)
denomination_factor = 1 / 100 = 0.01

Refund: ₹100.00 = 10000 paise
Converted: 10000 * 1.6 * 0.01 = 160 yen ✅
```

**Case 3: INR (2 decimals) → KWD (3 decimals)**
```
payment_currency_denomination = 100 (paise)
gateway_currency_denomination = 1000 (fils)
denomination_factor = 1000 / 100 = 10.0

Refund: ₹100.00 = 10000 paise
Converted: 10000 * 0.004 * 10.0 = 400 fils = 0.400 KWD ✅
```

### When This Applies

**Triggers**:
- DCC refunds with different decimal places between currencies
- Calculation in `GetGatewayAmountForDccPayment()`
- Line 466: `denominationFactor, _ := utils.GetGatewayCurrencyDenominationFactor(ctx, payment)`

**Field Sources**:
- `gateway_currency_denomination`: From payment entity (set during payment creation)
- `payment_raw_currency_denomination`: From payment entity (merchant's base currency)

---

## High-Priority Pattern 4: LRS Merchant Handling

### The Pattern

**Location**: `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go:317-320`

```go
} else if isLRSMerchant {
    gatewayAmount, _ = utils.ConvertToInt(ctx, paymentMeta[refundUtils.GatewayAmount])
    gatewayCurrency = utils.ConvertInterfaceToString(paymentMeta[refundUtils.GatewayCurrency])
    paymentGatewayAmount, _ = utils.ConvertToInt(ctx, paymentMeta[refundUtils.GatewayAmount])
```

### Why Different from DCC?

**LRS (Liberalized Remittance Scheme)**:
- Used for outward remittances (education fees, foreign investments, etc.)
- Gateway amount is **pre-calculated** by payments-cross-border
- No recalculation needed in scrooge (just use payment_meta values)

**DCC (Dynamic Currency Conversion)**:
- Used for international card payments
- Gateway amount **must be recalculated** based on refund amount
- Formula applies markup and denomination factor

### LRS Detection

**Location**: `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go:296`

```go
isLRSMerchant, _ := refundUtils.CheckLRSMerchantFeaturesFromDcs(ctx,
    utils.ConvertInterfaceToString(entities[constants.ExtractPaymentEntity].StringMap.Value[constants.MerchantID]))
```

**Checks**: Merchant features for LRS enablement via DCS (Data Config Service).

### When This Applies

**Triggers**:
- Merchant has LRS feature enabled
- International payments with pre-calculated gateway amounts
- Refunds for education fees, remittances, foreign investments

**Does NOT Apply**:
- Regular DCC payments (must recalculate)
- Domestic payments (no gateway currency)

---

## High-Priority Pattern 5: Base Amount Consistency

### The Pattern

**Location**: `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go:83-85, 104`

```go
// Calculate refund base amount
params[models.AttributeBaseAmount], ierr = refundUtils.GetBaseAmount(ctx, input)
if ierr != nil {
    return nil, ierr
}

// Store payment's base amount for comparison
params[models.AttributePaymentBaseAmount], _ = utils.ConvertFloat64ToInt64(
    entities[constants.ExtractPaymentEntity].StringMap.Value[constants.BaseAmount])
```

### Why Both Base Amounts?

**Refund Base Amount**: Calculated for this specific refund in merchant's base currency (INR).
**Payment Base Amount**: Original payment's base amount (for validation and comparison).

**Example**:
```
Payment: $100 USD → ₹8,300 INR base amount (rate: 83.00)

Partial Refund 1: $50 USD
- Refund base amount: ₹4,150 (calculated at current rate)
- Payment base amount: ₹8,300 (stored for comparison)

Partial Refund 2: $50 USD
- Refund base amount: ₹4,150 (calculated at current rate)
- Payment base amount: ₹8,300 (stored for comparison)

Validation: Sum of refund base amounts ≤ Payment base amount
₹4,150 + ₹4,150 = ₹8,300 ≤ ₹8,300 ✅
```

### When This Applies

**Triggers**:
- All cross-border refunds (DCC, MCC, LRS)
- Partial refund validation
- Base amount recalculation in refund flow

**Field Usage**:
- `AttributeBaseAmount`: Refund's base amount (calculated)
- `AttributePaymentBaseAmount`: Payment's base amount (from payment entity)

---

## Detection Patterns for Code Review

### Pattern 1: Math.Floor vs Math.Ceil in Refunds

**🚨 CRITICAL ISSUE**

**Detection**:
```
IF file contains: refund, gateway_amount, forex_rate
AND contains: Math.Ceil() or math.Ceil()
THEN flag: "Refunds must use Math.Floor, not Math.Ceil"
```

**Wrong**:
```go
// ❌ WRONG - Using Ceil in refunds causes money leaks
convertedAmount := refundAmount * forexRate * denominationFactor
gatewayAmount = int64(math.Ceil(convertedAmount + (markUpPercent*convertedAmount)/100))
```

**Correct**:
```go
// ✅ CORRECT - Using Floor prevents money leaks
convertedAmount := refundAmount * forexRate * denominationFactor
gatewayAmount = int64(math.Floor(convertedAmount + (markUpPercent*convertedAmount)/100))
```

**Why**: Partial refunds with Ceil can sum to more than payment amount.

**Where to Check**:
- `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go:469`
- Any new refund calculation functions

---

### Pattern 2: Three-Decimal Currency Rounding Missing

**⚠️ HIGH-PRIORITY**

**Detection**:
```
IF DCC refund gateway amount calculated
AND gateway_currency is KWD, OMR, or BHD
AND RoundDownGatewayAmountIfApplicable() NOT called
THEN flag: "Missing three-decimal currency rounding"
```

**Wrong**:
```go
// ❌ WRONG - Missing rounding for 3-decimal currencies
gatewayAmount = GetGatewayAmountForDccPayment(ctx, paymentMeta, input, payment)
// Directly use gatewayAmount (may have non-zero last digit)
```

**Correct**:
```go
// ✅ CORRECT - Apply rounding for 3-decimal currencies
gatewayAmount = GetGatewayAmountForDccPayment(ctx, paymentMeta, input, payment)
gatewayAmount = RoundDownGatewayAmountIfApplicable(ctx, paymentMeta, input, gatewayAmount)
```

**Why**: Card networks reject transactions with non-zero last digit for KWD/OMR/BHD.

**Where to Check**:
- `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go:322-323`
- Any new DCC refund calculation code

---

### Pattern 3: Denomination Factor Not Applied

**⚠️ HIGH-PRIORITY**

**Detection**:
```
IF DCC refund calculation
AND forex_rate used
AND denominationFactor NOT retrieved or NOT applied
THEN flag: "Missing denomination factor in forex calculation"
```

**Wrong**:
```go
// ❌ WRONG - Not using denomination factor
convertedAmount := refundAmount * forexRate
gatewayAmount = int64(math.Floor(convertedAmount + (markUpPercent*convertedAmount)/100))
```

**Correct**:
```go
// ✅ CORRECT - Include denomination factor
denominationFactor, _ := utils.GetGatewayCurrencyDenominationFactor(ctx, payment)
convertedAmount := refundAmount * forexRate * denominationFactor
gatewayAmount = int64(math.Floor(convertedAmount + (markUpPercent*convertedAmount)/100))
```

**Why**: Without denomination factor, conversions fail for 0-decimal (JPY) and 3-decimal (KWD) currencies.

**Where to Check**:
- `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go:466-467`
- Any new forex conversion functions

---

### Pattern 4: LRS Using DCC Calculation

**⚠️ HIGH-PRIORITY**

**Detection**:
```
IF isLRSMerchant = true
AND GetGatewayAmountForDccPayment() called
THEN flag: "LRS should use pre-calculated gateway_amount, not DCC formula"
```

**Wrong**:
```go
// ❌ WRONG - Recalculating gateway amount for LRS
} else if isLRSMerchant {
    gatewayAmount = GetGatewayAmountForDccPayment(ctx, paymentMeta, input, payment)
```

**Correct**:
```go
// ✅ CORRECT - Use pre-calculated value from payment_meta
} else if isLRSMerchant {
    gatewayAmount, _ = utils.ConvertToInt(ctx, paymentMeta[refundUtils.GatewayAmount])
    gatewayCurrency = utils.ConvertInterfaceToString(paymentMeta[refundUtils.GatewayCurrency])
```

**Why**: LRS gateway amounts are pre-calculated by payments-cross-border with special rates.

**Where to Check**:
- `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go:317-320`
- Any new LRS refund handling code

---

### Pattern 5: Base Amount Not Validated

**⚠️ HIGH-PRIORITY**

**Detection**:
```
IF refund processing
AND base_amount calculated
AND payment_base_amount NOT retrieved or NOT compared
THEN flag: "Missing base amount validation against payment"
```

**Wrong**:
```go
// ❌ WRONG - Not storing payment's base amount for validation
params[models.AttributeBaseAmount], ierr = refundUtils.GetBaseAmount(ctx, input)
// Missing: params[models.AttributePaymentBaseAmount] = ...
```

**Correct**:
```go
// ✅ CORRECT - Store both for validation
params[models.AttributeBaseAmount], ierr = refundUtils.GetBaseAmount(ctx, input)
if ierr != nil {
    return nil, ierr
}
params[models.AttributePaymentBaseAmount], _ = utils.ConvertFloat64ToInt64(
    entities[constants.ExtractPaymentEntity].StringMap.Value[constants.BaseAmount])
```

**Why**: Need to validate sum of refund base amounts doesn't exceed payment base amount.

**Where to Check**:
- `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go:83-85, 104`
- Refund validation logic

---

## Testing Checklist

### DCC Refund Calculations

- [ ] **Full DCC Refund**: Gateway amount equals payment gateway amount
- [ ] **Partial DCC Refund**: Gateway amount uses Math.Floor (not Ceil)
- [ ] **Multiple Partial Refunds**: Sum of gateway amounts ≤ payment gateway amount
- [ ] **Forex Rate Applied**: Correct forex_rate from payment_meta used
- [ ] **Markup Applied**: DCC markup percentage applied after conversion
- [ ] **Denomination Factor**: Applied for 3-decimal currencies (KWD, OMR, BHD)

### Three-Decimal Currency Rounding

- [ ] **KWD Refund**: Last digit rounded down to 0 (e.g., 1234 → 1230)
- [ ] **OMR Refund**: Last digit rounded down to 0
- [ ] **BHD Refund**: Last digit rounded down to 0
- [ ] **USD Refund**: No rounding (2-decimal currency)
- [ ] **JPY Refund**: No rounding (0-decimal currency)

### LRS Refunds

- [ ] **LRS Merchant**: Uses pre-calculated gateway_amount from payment_meta
- [ ] **LRS Merchant**: Does NOT call `GetGatewayAmountForDccPayment()`
- [ ] **Non-LRS Merchant**: Calculates gateway amount for DCC

### Base Amount Validation

- [ ] **Refund Base Amount**: Calculated correctly in merchant's base currency
- [ ] **Payment Base Amount**: Retrieved from payment entity
- [ ] **Validation**: Sum of refund base amounts ≤ payment base amount
- [ ] **Partial Refunds**: Each refund's base amount validated independently

### Test Scenarios

**Test 1: DCC INR → USD Refund**
```
Payment: ₹424,446 → $56.16 USD (rate: 83.00, markup: 8%)
Full Refund: ₹424,446 → $56.15 USD (Floor, not Ceil)
Expected: gatewayAmount = 5615 (1 cent less than payment)
```
Reference: `Test_RefundCreateDccINRtoUSD` at line 1055

**Test 2: DCC USD → INR Refund**
```
Payment: $50.00 → ₹4,288.57 INR (rate: 77.83, markup: 8%)
Full Refund: $50.00 → ₹4,288.56 INR (Floor, not Ceil)
Expected: gatewayAmount = 428856 (1 paisa less than payment)
```
Reference: `Test_RefundCreateDCCUSDtoINR` at line 1083

**Test 3: Three-Decimal Currency Rounding**
```
Payment: ₹100 → 0.404 KWD (3-decimal)
Calculated: 404 fils
After rounding: 400 fils (last digit → 0)
Expected: gatewayAmount = 400
```

---

## Code Locations Reference

### Core DCC Refund Functions

**`GetGatewayAmountAndCurrency()`**
- File: `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go`
- Line: 289-363
- Purpose: Main dispatcher for gateway amount calculation based on payment type
- Triggers: DCC (line 321), LRS (line 317), cardless_emi (line 306), etc.

**`GetGatewayAmountForDccPayment()`**
- File: `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go`
- Line: 456-471
- Purpose: Calculate gateway amount for DCC refunds using Math.Floor
- Uses: forex_rate, dcc_mark_up_percent, denomination_factor

**`RoundDownGatewayAmountIfApplicable()`**
- File: `/scrooge/app/services/internals/refund/create_v2_prepare_refund_params.go`
- Line: 483-494
- Purpose: Round down gateway amount for 3-decimal currencies (KWD, OMR, BHD)
- Logic: `gatewayAmount = (gatewayAmount / 10) * 10`

### Utility Functions

**`GetGatewayCurrencyDenominationFactor()`**
- File: `/scrooge/app/utils/utils.go`
- Line: 1201-1216
- Purpose: Calculate denomination factor for currency conversion
- Formula: `gateway_currency_denomination / payment_raw_currency_denomination`

**`IsLastDigitZeroCurrencies()`**
- File: `/scrooge/app/services/internals/refund/utils/currency.go`
- Line: 15-17
- Purpose: Check if currency requires last-digit-zero rounding (KWD, OMR, BHD)
- Returns: `true` for 3-decimal currencies

**`CalculateChargeWithForexRate()`**
- File: `/scrooge/app/utils/forex/forex.go`
- Line: 25-34
- Purpose: Generic forex rate calculation with denomination factors
- Uses: Math.Ceil() for general conversions

### Test Cases

**`Test_RefundCreateDccINRtoUSD`**
- File: `/scrooge/app/services/internals/refund/create_new_refund_v2_test.go`
- Line: 1055-1081
- Tests: Full DCC refund from INR to USD
- Validates: Gateway amount = 5615 (payment: 5616)

**`Test_RefundCreateDCCUSDtoINR`**
- File: `/scrooge/app/services/internals/refund/create_new_refund_v2_test.go`
- Line: 1083-1111
- Tests: Full DCC refund from USD to INR
- Validates: Gateway amount = 428856 (payment: 428857)

---

## Integration with Other Services

### Scrooge → Payments-Cross-Border

**Not Direct Integration**: Scrooge does NOT call payments-cross-border for refund calculations.

**Why**: Refunds use payment_meta fields (forex_rate, dcc_mark_up_percent) that were set during payment authorization by payments-cross-border. Scrooge applies the same rates from payment creation time.

### Scrooge → Payment Entity

**Field Dependencies**:
- `payment_meta.forex_rate` - Forex rate from payment authorization
- `payment_meta.dcc_mark_up_percent` - DCC markup percentage
- `payment_meta.gateway_amount` - Original payment gateway amount
- `payment_meta.gateway_currency` - Gateway currency (USD, EUR, etc.)
- `payment.base_amount` - Payment base amount in merchant currency
- `payment.gateway_currency_denomination` - Gateway currency denomination (100, 1000, etc.)
- `payment.payment_raw_currency_denomination` - Payment currency denomination

---

## Key Differences: Payment vs Refund

| Aspect | Payment (payments-cross-border/payments-card) | Refund (scrooge) |
|--------|----------------------------------------------|------------------|
| **Rounding Method** | Math.Ceil() | Math.Floor() |
| **Why** | Favor customer, ensure full amount charged | Prevent money leaks from partial refunds |
| **3-Decimal Rounding** | Round UP to nearest 10 | Round DOWN to nearest 10 |
| **DCC Calculation** | Calculate forex charges, create DCC | Use pre-calculated forex_rate from payment |
| **Denomination Factor** | Applied during conversion | Applied during refund conversion |
| **Rate Source** | Real-time from payments-cross-border | Stored in payment_meta from payment time |
| **Markup/Markdown** | Applied during authorization | Re-applied from payment_meta values |

---

## Summary

Scrooge's cross-border refund handling follows these principles:

1. **Money Leak Prevention**: Use Math.Floor() instead of Math.Ceil() to prevent partial refunds from summing to more than payment amount
2. **Network Compliance**: Round down 3-decimal currencies (KWD, OMR, BHD) to nearest 10
3. **Denomination Accuracy**: Apply denomination factors for currencies with different decimal places
4. **Rate Consistency**: Use forex rates and markup percentages from payment creation time (stored in payment_meta)
5. **LRS Special Handling**: Don't recalculate gateway amounts for LRS (use pre-calculated values)
6. **Base Amount Validation**: Ensure sum of refund base amounts doesn't exceed payment base amount

**Critical Review Focus**: When reviewing scrooge refund code, verify that Math.Floor() is used (NOT Math.Ceil()), three-decimal rounding is applied, and denomination factors are included in calculations.

---

**File Version**: 1.0.0
**Last Updated**: 2026-02-19
**Service**: scrooge (Collections & Refunds)
**Scope**: DCC, LRS, MCC refund processing


---

# Wallet Payments (Apple Pay / Google Pay) Cross-Border Patterns

This document explains cross-border specific patterns for wallet payments (Apple Pay, Google Pay) across pg-router and payments-card services.

## Overview

**Services Involved**:
- `pg-router` - DCC info caching and routing logic
- `payments-card` - Wallet payment processing with DCC

**Key Difference from Card Payments**:
- Wallet payments are "skip-initiate" providers - they skip the payment initiate step
- DCC (Dynamic Currency Conversion) info must be cached during payment creation and retrieved later
- Risk provider tokens are handled differently due to the skip-initiate flow

---

## Critical Pattern 1: DCC Info Caching for Wallet Payments

**Problem**: Apple Pay/Google Pay skip the initiate step, but DCC currency selection happens before payment creation. How to preserve DCC info across the flow?

### Solution: Cache-Based DCC Info Passing

**Location**: `pg-router/internal/payments/core/create.go:storeDCCInfoForSkipInitiateProviders()` (line 1166)

**Flow**:

1. **Payment Create** (with DCC selection):
   - Request includes: `dcc_currency`, `currency_request_id`, `amount`, `currency`
   - After payment status = "created"
   - Check if provider skips initiate (`shouldInitiatePayment()` returns false)
   - Store DCC info in Redis cache

2. **Cache Storage**:
   ```go
   // Extract DCC fields from request
   dccInfo := map[string]interface{}{
       "currency":            requestMap["currency"],
       "amount":              requestMap["amount"],
       "dcc_currency":        requestMap["dcc_currency"],
       "currency_request_id": requestMap["currency_request_id"],
   }

   // Store in cache with 12-minute TTL
   crossBorderExportService.StoreProviderDCCInfoInCache(ctx, paymentId, dccInfo)
   ```

3. **Cache Key**: `provider_dcc_info_{paymentId}` (with fallback to `apple_dcc_info_{paymentId}`)

4. **Retrieval During Authorize**:
   - `pg-router/internal/payments/routing/controller.go` (line 2617-2639)
   - Before partner pay authorization
   - Retrieve cached DCC info
   - Populate `paymentAuthorizeRequest` with DCC fields

**Critical Check**:

```go
// ❌ WRONG - Storing DCC info for all providers
if cpsResponseStruct.Data.Payment.Status == "created" {
    storeDCCInfoForSkipInitiateProviders(...)
}

// ✅ CORRECT - Only for skip-initiate providers
if cpsResponseStruct.Data.Payment.Status == "created" &&
   !shouldInitiatePayment() && !isS2S {
    storeDCCInfoForSkipInitiateProviders(...)
}
```

**Why Critical**:
- Without caching, DCC currency selection is lost → customer charged in wrong currency
- Sending DCC info for non-skip-initiate providers causes duplicate DCC calls
- Cache miss means payment proceeds without DCC → incorrect forex rates applied

---

## Critical Pattern 2: User Risk Provider Token Handling

**Problem**: Wallet payments need risk assessment, but the flow is different from regular card payments.

### Risk Token Flow for Wallets

**Location**: `pg-router/internal/coproto/pxb_helper.go:userRiskDetailProviders()` (line 41)

**Decision Logic**:

```go
// 1. Check if payment is international
if !isInternationalPayment() {
    return nil  // No risk providers needed for domestic
}

// 2. Check if risk token already collected (cached)
if cbExportClient.IsUserRiskProvidersTokenFoundInCache(ctx, paymentId) {
    return nil  // Skip loading risk provider JS - token already collected
}

// 3. Load risk provider JS via Splitz experiments
// (Fingerprint, Riskified, Sardine)
```

**Cache Flow**:

1. **Token Storage** (during payment create):
   - `pg-router/internal/payments/core/create.go:setUserRiskProvidersToken()` (line 1201)
   - Extract token from request or cache
   - Store in cache: `user_risk_providers_token_{paymentId}` with 12-min TTL
   - Pass to CPS in request map

2. **Token Check** (during coproto response):
   - Check cache for existing token
   - If found: skip loading risk JS (performance optimization)
   - If not found: load risk provider JavaScript

**Critical Anti-Pattern**:

```go
// ❌ WRONG - Always loading risk provider JS
func userRiskDetailProviders(ctx, paymentId) {
    // Load Fingerprint, Riskified, Sardine JS
    return buildRiskProviders()
}

// ✅ CORRECT - Check cache first
func userRiskDetailProviders(ctx, paymentId) {
    if cbExportClient.IsUserRiskProvidersTokenFoundInCache(ctx, paymentId) {
        return nil  // Already collected
    }
    return buildRiskProviders()
}
```

**Why Critical**:
- Loading unnecessary JS slows down checkout experience
- Risk token may already be collected from previous step in wallet flow
- Cache check prevents duplicate risk assessment

---

## Critical Pattern 3: Wallet Payment Gateway Amount/Currency

**Problem**: Wallet payments with DCC need to send correct amount/currency to risk/fraud systems.

**Pattern** (Same as regular DCC, from payments-card):

**Location**: `payments-card/pkg/cross_border/helper.go:getAmount/getCurrency()` (line 193-205)

```go
// Helper functions that prefer gateway amount/currency for DCC
func getAmount(payment common.IPayment) float64 {
    if utils.IsEmpty(payment.GetGatewayAmount()) {
        return float64(payment.GetAmount())
    }
    return float64(payment.GetGatewayAmount())  // ✅ Prefer gateway amount
}

func getCurrency(payment common.IPayment) string {
    if utils.IsEmpty(payment.GetGatewayCurrency()) {
        return payment.GetCurrency()
    }
    return payment.GetGatewayCurrency()  // ✅ Prefer gateway currency
}
```

**Usage for Wallets**:

```go
// ❌ WRONG - Using payment.Amount for wallet payment with DCC
riskRequest.Amount = payment.GetAmount()  // Wrong for Apple Pay with DCC

// ✅ CORRECT - Using helper that checks gateway fields
riskRequest.Amount = getAmount(payment)  // Correct for all payment types
```

**Why Critical for Wallets**:
- Wallet payments (Apple Pay/Google Pay) frequently use DCC
- Risk/fraud systems receive incorrect data if using payment.Amount instead of gateway_amount
- Validation failures occur when comparing wrong currency amounts

---

## High-Priority Pattern 1: Apple DCC Selector Form

**Problem**: Apple Pay has a specific DCC selector UI flow.

**Template**: `pg-router/resources/templates/appleDccSelectorForm.html.tmpl`

**Usage**:
- Renders DCC currency selection form for Apple Pay
- Displays exchange rates and currency options
- Collects `dcc_currency` and `currency_request_id` from user

**Critical Check**:

```go
// When generating Apple Pay DCC form, ensure:
// ✅ All available currencies from PXB are displayed
// ✅ Exchange rates are locked via currency_request_id
// ✅ Form submits to correct payment create endpoint
```

**Why Important**:
- Apple Pay has specific UX requirements for DCC
- Incorrect form rendering prevents DCC selection
- Missing currency_request_id causes rate mismatch

---

## High-Priority Pattern 2: Fallback Cache Key for Apple Pay

**Problem**: Legacy Apple Pay DCC info used different cache key.

**Implementation**: `pg-router/internal/cross_border_export/service.go:GetProviderDCCInfoFromCache()` (line 77-82)

```go
// Primary key lookup
cacheKey := getProviderDCCInfoCacheKey(paymentId)  // "provider_dcc_info_{id}"
res, err := cache.Get(ctx, cacheKey)
if err != nil {
    // ✅ Fallback to legacy Apple Pay key
    cacheKey := getAppleDCCInfoCacheKey(paymentId)  // "apple_dcc_info_{id}"
    res, err = cache.Get(ctx, cacheKey)
}
```

**Why Important**:
- Supports backward compatibility with old Apple Pay flows
- Prevents cache misses for in-flight Apple Pay payments
- Gradual migration from old to new cache key structure

---

## High-Priority Pattern 3: Empty DCC Info Handling

**Problem**: Not all wallet payments require DCC (domestic wallets, merchant choice).

**Check**: `pg-router/internal/payments/core/create.go` (line 1192)

```go
// Extract DCC fields
dccInfo := extractDCCFieldsFromRequest(requestMap)

// ❌ WRONG - Storing empty DCC info
crossBorderExportService.StoreProviderDCCInfoInCache(ctx, paymentId, dccInfo)

// ✅ CORRECT - Check if DCC info is empty first
if len(dccInfo) == 0 {
    logger.Info("DCC info is empty, skipping cache storage")
    return
}
crossBorderExportService.StoreProviderDCCInfoInCache(ctx, paymentId, dccInfo)
```

**Why Important**:
- Prevents storing empty data in cache (wastes memory)
- Avoids cache pollution with null/empty values
- Differentiates between "no DCC" vs "DCC cache miss"

---

## Detection Patterns for Code Review

### Pattern 1: Missing DCC Cache for Skip-Initiate Providers

**🚨 CRITICAL**

```go
// ❌ WRONG - Not caching DCC info for wallet payment
if cpsResponseStruct.Data.Payment.Status == "created" {
    // Missing storeDCCInfoForSkipInitiateProviders call
}

// ✅ CORRECT
if cpsResponseStruct.Data.Payment.Status == "created" &&
   !shouldInitiatePayment() && !isS2S && paymentRequest.Id != nil {
    storeDCCInfoForSkipInitiateProviders(ctx, requestMap, *paymentRequest.Id)
}
```

**Where to Check**:
- Payment create success handling in pg-router
- Any new wallet payment method integration

---

### Pattern 2: Incorrect DCC Info Retrieval

**⚠️ HIGH-PRIORITY**

```go
// ❌ WRONG - Not retrieving cached DCC info before authorize
paymentAuthorizeRequest := buildAuthorizeRequest(payment)

// ✅ CORRECT
cachedDCCInfo := cbExportClient.GetProviderDCCInfoFromCache(ctx, paymentId)
if cachedDCCInfo != nil {
    paymentAuthorizeRequest.CurrencyRequestId = cachedDCCInfo["currency_request_id"]
    paymentAuthorizeRequest.DccCurrency = cachedDCCInfo["dcc_currency"]
    // ... populate other fields
}
```

**Where to Check**:
- Payment authorization controller in pg-router
- Partner pay flow processing

---

### Pattern 3: Missing Risk Token Cache Check

**⚠️ HIGH-PRIORITY**

```go
// ❌ WRONG - Always loading risk provider JS
if payment.IsInternational() {
    return buildRiskProviders()  // Always loads JS
}

// ✅ CORRECT
if payment.IsInternational() {
    if cbExportClient.IsUserRiskProvidersTokenFoundInCache(ctx, paymentId) {
        return nil  // Skip loading - already collected
    }
    return buildRiskProviders()
}
```

**Where to Check**:
- Coproto response generation
- Checkout page rendering

---

### Pattern 4: Gateway Amount for Wallet DCC Payments

**🚨 CRITICAL**

```go
// ❌ WRONG - Direct field access for wallet payment
if payment.GetWallet() == "applepay" || payment.GetWallet() == "googlepay" {
    riskRequest.Amount = payment.GetAmount()  // Wrong for DCC
}

// ✅ CORRECT - Using helper functions
riskRequest.Amount = getAmount(payment)      // Handles DCC correctly
riskRequest.Currency = getCurrency(payment)  // Handles DCC correctly
```

**Where to Check**:
- Risk evaluation calls in payments-card
- Fraud detection API calls
- Any external service integration

---

## Testing Checklist

### For Wallet DCC Payments

- [ ] DCC info cached during payment create for skip-initiate providers
- [ ] DCC info retrieved from cache during authorize
- [ ] Cache key includes payment ID
- [ ] Cache TTL is 12 minutes (sufficient for wallet flow)
- [ ] Empty DCC info not stored in cache
- [ ] Fallback to Apple Pay legacy key works

### For Wallet Risk Assessment

- [ ] Risk token cached if present in request
- [ ] Risk token retrieved from cache when available
- [ ] Risk JS not loaded if token already collected
- [ ] International wallet payments get risk check
- [ ] Domestic wallet payments skip risk providers

### For Wallet Payment Fields

- [ ] Gateway amount used for DCC wallet payments
- [ ] Gateway currency used for DCC wallet payments
- [ ] `forex_applied` flag set when DCC applied to wallet payment
- [ ] Helper functions used instead of direct field access

---

## Common Issues

### Issue 1: DCC Lost for Apple Pay

**Symptom**: Customer selects DCC currency, but payment processes in merchant currency

**Root Cause**: DCC info not cached or cache miss during authorize

**Fix**:
1. Verify `storeDCCInfoForSkipInitiateProviders()` called after payment create
2. Check cache TTL hasn't expired (12 minutes)
3. Ensure payment ID used in cache key is consistent

---

### Issue 2: Duplicate Risk Assessment

**Symptom**: Risk provider JS loaded twice, slowing checkout

**Root Cause**: Not checking cache for existing risk token

**Fix**:
1. Add `IsUserRiskProvidersTokenFoundInCache()` check before loading JS
2. Return nil/empty if token already collected

---

### Issue 3: Wrong Amount Sent to Risk System

**Symptom**: Risk score incorrect for wallet DCC payments

**Root Cause**: Using `payment.Amount` instead of `gateway_amount` for DCC

**Fix**:
1. Use `getAmount(payment)` helper function
2. Use `getCurrency(payment)` helper function
3. Never directly access Amount/Currency fields for international payments

---

## Related Documentation

- [MCC/DCC Lifecycle Flows](#mccdcc-lifecycle-flows-and-currency-states) - General DCC flow patterns
- [Payments-Card Patterns](#payments-card-cross-border-patterns) - Card DCC patterns apply to wallets too
- [PG-Router Cross-Border Export Flows](../../pg-router/.agents/skills/repo-skill/modules/domain/cross_border_export/flows.md) - Detailed cache flow documentation

---

## Code Locations Reference

### PG-Router

- **DCC Info Storage**: `internal/payments/core/create.go:1166-1199`
- **DCC Info Retrieval**: `internal/payments/routing/controller.go:2617-2640`
- **Risk Token Storage**: `internal/payments/core/create.go:1201-1209`
- **Risk Token Check**: `internal/coproto/pxb_helper.go:41-53`
- **Cross-Border Export Service**: `internal/cross_border_export/service.go`
- **Cache Keys**: `internal/cross_border_export/helper.go`
- **Apple DCC Form**: `resources/templates/appleDccSelectorForm.html.tmpl`

### Payments-Card

- **Gateway Amount Helpers**: `pkg/cross_border/helper.go:193-205`
- **CFB Fee Check**: `pkg/cross_border/helper.go:185-191` (applies to wallets)
- **DCC Creation**: `pkg/cross_border/cross_border.go:264-291`

---

**Version**: 1.0.0
**Last Updated**: 2026-03-13
**Maintainer**: Cross-Border Code Review Skill


---

# Skip3DS Cross-Border Patterns

This document explains skip3DS (Skip 3D Secure) flows for international card payments across shield and payments-card services.

## Overview

**Services Involved**:
- `shield` - Skip3DS rule engine evaluation (cross-border module)
- `payments-card` - Skip3DS integration and authorization retry logic

**What is Skip3DS**:
- Mechanism to bypass 3D Secure authentication for low-risk international card transactions
- Reduces friction for repeat customers and trusted merchants
- Based on risk scoring via Sardine, merchant risk profiles, and chargeback protection

---

## Critical Pattern 1: Skip3DS Rule Engine Evaluation

**Problem**: How to decide whether to skip 3DS for an international payment?

### Rule Engine Architecture

**Location**: `shield/app/crossBorder/app/services/skip3DS/core.go:EvaluateRuleEngine()` (line 44)

**Flow**:

1. **Build Supporting Entities**:
   - Card entity (IIN, country, enrollment status, fingerprint)
   - Payment entity (base amount, gateway)
   - Merchant entity (ID, name, creation date, activation date)

2. **Build Evaluating Entities**:
   - Derived params (velocity, repeat customer, direct params)
   - Extracted from payment input

3. **Evaluate Default Rules**:
   - Default rejection rules (apply to all merchants)
   - Default scoring rules (risk score calculation)

4. **Evaluate Merchant Risk Profile Rules**:
   - High-risk merchant rules (stricter criteria)
   - Medium-risk merchant rules (moderate criteria)
   - Low-risk merchant rules (relaxed criteria)
   - **Decision Point**: Splitz experiment determines merchant risk profile

5. **Decision Maker Evaluation**:
   - Combines rejection + scoring results
   - Returns boolean: `true` = skip 3DS, `false` = enforce 3DS

6. **Sardine Decision Maker** (ALL payments):
   - Evaluates Sardine-specific rejection rules
   - **Decision Point**: Splitz experiment `HonorSardineMerchantDecisionMakerResponse`
   - If experiment = "on" → honor Sardine decision
   - If experiment = "control" → log but don't honor

7. **Final Decision**:
   - If main decision maker returns `true` → skip 3DS
   - If main decision maker returns `false` AND Sardine returns `true` AND experiment = "on" → skip 3DS
   - Otherwise → enforce 3DS

**Critical Check**:

```go
// ❌ WRONG - Skipping 3DS without rule engine evaluation
if payment.IsInternational() {
    skipThreeDS = true  // Dangerous!
}

// ✅ CORRECT - Evaluate rule engine
skip3DSEngine := NewSkip3DSEngine()
skipThreeDS, err := skip3DSEngine.EvaluateRuleEngine(ctx, payload, derivedParams)
if err != nil {
    skipThreeDS = false  // Safe fallback - enforce 3DS
}
```

**Why Critical**:
- Skipping 3DS without evaluation exposes merchant to chargeback risk
- Not evaluating Sardine rules misses important fraud signals
- Missing experiment checks causes inconsistent behavior

---

## Critical Pattern 2: Skip3DS Response Caching

**Problem**: Shield evaluates skip3DS during pre-auth leg. How to pass result to payments-card for authorization retry?

### Cache-Based Result Passing

**Location**: `payments-card/internal/workflow/pay/pxb_helper.go:shouldRetryAuthorization()` (line 63-71)

**Flow**:

1. **Shield Evaluation** (pre-auth):
   - Shield evaluates skip3DS rules
   - Result stored in Redis: `skip3ds_risk_check_{paymentId}`

2. **Payments-Card Retrieval** (authorization):
   - Check context: `ctx.Get(constants.Skip3DSRuleEngineEvaluation)`
   - Fallback to cache: `provider.EntityStore.Fetch(ctx, paymentId, constants.Skip3DSRiskCheckRedisGroup)`

3. **Authorization Retry Decision**:
   - If skip3DS = true AND authorization failed with specific error codes
   - Retry authorization without 3DS

**Critical Check**:

```go
// ❌ WRONG - Not checking cache for skip3DS result
if authorizationFailed {
    return false  // Don't retry
}

// ✅ CORRECT - Check cache and retry if skip3DS allowed
if splitzResp := getSkip3DSExperiment(); splitzResp.Name == "on" {
    if skip3DS, ok := ctx.Get(constants.Skip3DSRuleEngineEvaluation); ok {
        return handleNewRuleEngineResponse(ctx, cast.ToBool(skip3DS), ...)
    }
    if skip3dsResponse, err := cache.Fetch(ctx, paymentId, "skip3ds_risk_check"); err == nil {
        return handleNewRuleEngineResponse(ctx, cast.ToBool(skip3dsResponse), ...)
    }
}
```

**Why Critical**:
- Missing cache check means skip3DS evaluation is wasted
- Authorization retry logic depends on accurate skip3DS result
- Cache miss causes 3DS to be enforced even when skip is allowed

---

## Critical Pattern 3: Authorization Retry with Skip3DS

**Problem**: Authorization fails with 3DS. Should we retry without 3DS based on skip3DS evaluation?

### Retry Decision Logic

**Location**: `payments-card/internal/workflow/pay/pxb_helper.go:shouldRetryAuthorization()` (line 40)

**Conditions (ALL must be true)**:

1. ✅ Payment is international
2. ✅ Not a cross-border network tokenized payment
3. ✅ Skip3DS authz retry experiment is "on"
4. ✅ Skip3DS rule engine evaluation = true (from cache)
5. ✅ Authorization failed with retriable error code
6. ✅ Not already retried once (check context key)
7. ✅ Card enrollment status is NOT "not_enrolled" or "unknown"

**Implementation**:

```go
func shouldRetryAuthorization(ctx, actionRequest, authenticationEntity, authorizationEntity) bool {
    payment, _ := actionRequest.GetPayment()
    merchant, _ := actionRequest.GetMerchant()
    authzErrorCode := authorizationEntity.ErrorCode

    // 1. Check basic conditions
    if !payment.IsInternational() || utils.IsEmpty(authzErrorCode) {
        return false
    }

    // 2. Check if cross-border network tokenized (skip retry)
    if pxb_utils.IsCBNetworkTokenised(...) {
        return false
    }

    // 3. Check experiment
    if !considerAuthzRetryViaNon3ds(ctx, payment.GetID()) {
        return false
    }

    // 4. Check if already retried
    if _, ok := ctx.Get(constants.RetryAuthorizationContextKey); ok {
        return false
    }

    // 5. Get skip3DS result from context or cache
    if skip3DS, ok := ctx.Get(constants.Skip3DSRuleEngineEvaluation); ok {
        return handleNewRuleEngineResponse(ctx, cast.ToBool(skip3DS), ...)
    }

    if skip3dsResponse, err := cache.Fetch(ctx, paymentId, "skip3ds_risk_check"); err == nil {
        return handleNewRuleEngineResponse(ctx, cast.ToBool(skip3dsResponse), ...)
    }

    return false
}
```

**Critical Anti-Pattern**:

```go
// ❌ WRONG - Retrying without checking skip3DS evaluation
if authorizationFailed && payment.IsInternational() {
    retryWithoutThreeDS()  // Dangerous - no risk check!
}

// ✅ CORRECT - Check skip3DS evaluation first
if authorizationFailed && payment.IsInternational() {
    if skip3DSAllowed := getSkip3DSResult(ctx, paymentId); skip3DSAllowed {
        retryWithoutThreeDS()
    }
}
```

**Why Critical**:
- Retrying without skip3DS check bypasses risk evaluation
- Could expose merchant to fraud/chargebacks
- Experiment control is essential for gradual rollout

---

## Critical Pattern 4: Merchant Risk Profile Classification

**Problem**: Different merchants have different risk profiles. How to apply appropriate skip3DS rules?

### Risk Profile Experiments

**Location**: `shield/app/crossBorder/app/services/skip3DS/core.go` (lines 361-392)

**Three Risk Profiles**:

1. **High-Risk Merchants**:
   - Splitz experiment: `EvaluateMerchantWithHighRiskProfile`
   - Stricter skip3DS criteria
   - Higher rejection thresholds

2. **Medium-Risk Merchants**:
   - Splitz experiment: `EvaluateMerchantWithMediumRiskProfile`
   - Moderate skip3DS criteria
   - Balanced rejection thresholds

3. **Low-Risk Merchants**:
   - Splitz experiment: `EvaluateMerchantWithLowRiskProfile`
   - Relaxed skip3DS criteria
   - Lower rejection thresholds

**Evaluation Flow**:

```go
// Check merchant risk profile via Splitz
if evaluateMerchantWithHighRiskProfile(ctx, paymentId, merchantId) {
    // Apply high-risk rules
    highRiskRejection, _ := high_risk_merchants.EvaluateHighRiskMerchantRejection(...)
    highRiskScoring, _ := high_risk_merchants.EvaluateHighRiskMerchantScoring(...)
} else if evaluateMerchantWithLowRiskProfile(ctx, paymentId, merchantId) {
    // Apply low-risk rules
    lowRiskRejection, _ := low_risk_merchants.EvaluateLowRiskMerchantRejection(...)
    lowRiskScoring, _ := low_risk_merchants.EvaluateLowRiskMerchantScoring(...)
} else if evaluateMerchantWithMediumRiskProfile(ctx, paymentId, merchantId) {
    // Apply medium-risk rules
    mediumRiskRejection, _ := medium_risk_merchants.EvaluateMediumRiskMerchantRejection(...)
    mediumRiskScoring, _ := medium_risk_merchants.EvaluateMediumRiskMerchantScoring(...)
}
```

**Critical Check**:

```go
// ❌ WRONG - Applying same rules to all merchants
skipThreeDS := evaluateDefaultRules(payment)

// ✅ CORRECT - Apply risk-profile-specific rules
merchantRiskProfile := getMerchantRiskProfile(merchantId)
switch merchantRiskProfile {
case HighRisk:
    skipThreeDS = evaluateHighRiskRules(payment)
case MediumRisk:
    skipThreeDS = evaluateMediumRiskRules(payment)
case LowRisk:
    skipThreeDS = evaluateLowRiskRules(payment)
}
```

**Why Critical**:
- High-risk merchants need stricter controls to prevent chargebacks
- Low-risk merchants can have better conversion with relaxed rules
- Applying wrong profile causes either excessive friction or excessive risk

---

## High-Priority Pattern 1: Sardine Decision Maker Integration

**Problem**: Sardine provides real-time fraud signals. How to integrate into skip3DS decision?

### Sardine Evaluation Flow

**Location**: `shield/app/crossBorder/app/services/skip3DS/core.go` (lines 143-210)

**Implementation**:

```go
// Evaluate Sardine rules for ALL payments (not just specific risk profiles)
sardineMerchantRejectionResp, err := sardine_merchants.EvaluateSardineMerchantRejection(ctx, supportingEntities, evaluatingEntity, paymentId)

// Build decision maker input
sardineDecisionMakerInput := map[string]interface{}{
    "sardine_merchant_rejection_response": sardineMerchantRejectionResp,
}

// Evaluate Sardine decision maker
sardineDecisionMakerResp, err := decision_maker.EvaluateSardineDecisionMaker(ctx, supportingEntities, sardineDecisionMakerEvaluatingEntity, paymentId)

// Push event for monitoring
cbEvents.PushSardineDecisionMakerInitiatedEvent(ctx, &cbModels.Skip3DSEvaluationStartPayload{...})

// Honor Sardine response if main decision maker returned false
if finalDecisionMakerResp {
    return finalDecisionMakerResp, nil  // Main decision maker allows skip3DS
}

// Main decision maker denied, check if should honor Sardine
if sardineDecisionMakerResp {
    if honorSardineDecisionMakerResponse(ctx, paymentId, merchantId) {
        // Honor Sardine's allow decision
        cbEvents.PushSardineDecisionMakerHonoredEvent(ctx, ...)
        return sardineDecisionMakerResp, nil
    } else {
        // Log but don't honor (control group)
        cbEvents.PushSardineDecisionMakerHonoredEvent(ctx, ...)
    }
}

// Sardine not honored or denied, return main decision
return finalDecisionMakerResp, nil
```

**Why Important**:
- Sardine provides additional fraud signals beyond traditional rules
- Experiment-based rollout allows A/B testing of Sardine integration
- Events enable monitoring of Sardine decision impact

---

## High-Priority Pattern 2: Chargeback Protection Evaluation

**Problem**: For authorization retry, need to evaluate chargeback risk before skipping 3DS.

### Chargeback Protection Call

**Location**: `payments-card/internal/workflow/pay/pxb_helper.go:shouldRetryAuthorization()` (lines 110-122)

**Implementation**:

```go
// Fetch repeat customer flags
repeatCustomerFlags := steps.GetRepeatCustomerDetailsByCardFingerprint(ctx, merchantId, cardFingerprint)
steps.GetRepeatCustomerPaymentDetails(ctx, merchantId, cardCountry, repeatCustomerFlags)

// Get Cybersource score from context or cache
var cybersourceScore int32 = -1
if score, ok := ctx.Get(constants.CybersourceScore); ok {
    cybersourceScore = cast.ToInt32(score)
} else {
    // Fallback to cached Shield metadata
    cachedMetadata := steps.GetShieldMetadataForPayment(ctx, actionRequest)
    if cachedMetadata[constants.CybersourceScore] != nil {
        cybersourceScore = cast.ToInt32(cachedMetadata[constants.CybersourceScore])
    }
}

// Call PXB to evaluate chargeback protection
fields := populateRequestFieldsForChargebackProtection(
    constants.PurposeSmartAuthz,
    authenticationEntity.Enrolled,
    authzErrorCode,
    cybersourceScore,
)
res, err := cross_border.Get().EvaluateChargebackProtection(ctx, actionRequest, repeatCustomerFlags, fields)

// Check result
if !res.Success || !res.FilterResult || utils.IsEmpty(res.FilterAction.OverrideAuth) {
    return false  // Don't retry
}

// Retry allowed
return true
```

**Why Important**:
- Chargeback protection adds additional layer of risk assessment
- Repeat customer data helps identify trusted customers
- Cybersource score provides fraud signal

---

## High-Priority Pattern 3: Cross-Border Network Tokenization Skip

**Problem**: Network tokenized cross-border payments should not use skip3DS retry logic.

### Network Token Check

**Location**: `payments-card/internal/workflow/pay/pxb_helper.go:shouldRetryAuthorization()` (lines 49-51)

```go
// Check if cross-border network tokenized payment
if pxb_utils.IsCBNetworkTokenised(
    merchant.GetCountryCode(),
    actionRequest.Input.Card.GetCountry(),
    actionRequest.Input.Card.GetTokenised(),
    actionRequest.Input.Card.GetTokenIin(),
) {
    return false  // Skip retry logic for network tokens
}
```

**Why Important**:
- Network tokenized payments have different risk profiles
- Separate authorization logic for network tokens
- Prevents interference with network token flows

---

## Detection Patterns for Code Review

### Pattern 1: Missing Skip3DS Rule Engine Evaluation

**🚨 CRITICAL**

```go
// ❌ WRONG - Skipping 3DS without evaluation
if payment.IsInternational() && merchant.IsTrusted() {
    skipThreeDS = true  // No rule engine evaluation!
}

// ✅ CORRECT - Evaluate via Shield rule engine
skip3DSEngine := NewSkip3DSEngine()
payload := buildEvaluationPayload(payment, merchant, card)
derivedParams := buildDerivedParams(payment)
skipThreeDS, err := skip3DSEngine.EvaluateRuleEngine(ctx, payload, derivedParams)
if err != nil {
    skipThreeDS = false  // Safe fallback
}
```

**Where to Check**:
- Any code that decides whether to skip 3DS
- International payment processing flows
- Authorization logic in payments-card

---

### Pattern 2: Missing Skip3DS Cache Check for Retry

**⚠️ HIGH-PRIORITY**

```go
// ❌ WRONG - Retrying without checking skip3DS result
if authzFailed && payment.IsInternational() {
    retryWithoutThreeDS()
}

// ✅ CORRECT - Check cached skip3DS result
if authzFailed && payment.IsInternational() {
    // Check context first
    if skip3DS, ok := ctx.Get(constants.Skip3DSRuleEngineEvaluation); ok {
        if cast.ToBool(skip3DS) {
            retryWithoutThreeDS()
        }
        return
    }
    // Fallback to cache
    if skip3dsResponse, err := cache.Fetch(ctx, paymentId, "skip3ds_risk_check"); err == nil {
        if cast.ToBool(skip3dsResponse) {
            retryWithoutThreeDS()
        }
    }
}
```

**Where to Check**:
- Authorization retry logic
- Failed authorization handling
- 3DS bypass logic

---

### Pattern 3: Not Respecting Merchant Risk Profile

**⚠️ HIGH-PRIORITY**

```go
// ❌ WRONG - Same rules for all merchants
skipThreeDS := evaluateDefaultRules(payment)

// ✅ CORRECT - Risk-profile-specific rules
if evaluateMerchantWithHighRiskProfile(ctx, paymentId, merchantId) {
    // High-risk rules (stricter)
    rejectionResp, _ := high_risk_merchants.EvaluateHighRiskMerchantRejection(...)
    scoringResp, _ := high_risk_merchants.EvaluateHighRiskMerchantScoring(...)
} else if evaluateMerchantWithLowRiskProfile(ctx, paymentId, merchantId) {
    // Low-risk rules (relaxed)
    rejectionResp, _ := low_risk_merchants.EvaluateLowRiskMerchantRejection(...)
    scoringResp, _ := low_risk_merchants.EvaluateLowRiskMerchantScoring(...)
}
```

**Where to Check**:
- Shield rule engine evaluation
- Skip3DS decision logic
- Merchant-specific rule application

---

### Pattern 4: Missing Sardine Decision Maker Check

**⚠️ HIGH-PRIORITY**

```go
// ❌ WRONG - Not evaluating Sardine decision
finalDecision := evaluateMainDecisionMaker(...)
return finalDecision

// ✅ CORRECT - Check Sardine if main decision is false
finalDecision := evaluateMainDecisionMaker(...)
if finalDecision {
    return finalDecision  // Main decision allows
}

// Main decision denied, check Sardine
sardineDecision, _ := evaluateSardineDecisionMaker(...)
if sardineDecision && honorSardineDecisionMakerResponse(ctx, paymentId, merchantId) {
    // Honor Sardine's allow decision
    return sardineDecision
}

return finalDecision  // Return main decision
```

**Where to Check**:
- Skip3DS rule engine final decision
- Decision maker integration
- Sardine experiment handling

---

## Testing Checklist

### For Skip3DS Rule Engine

- [ ] Rule engine evaluation called for international payments
- [ ] Supporting entities properly built (card, payment, merchant)
- [ ] Evaluating entities include derived params
- [ ] Default rules evaluated for all payments
- [ ] Merchant risk profile rules applied based on Splitz experiment
- [ ] Decision maker combines all rule results
- [ ] Sardine decision maker evaluated for all payments
- [ ] Sardine experiment controls whether to honor Sardine result
- [ ] Events pushed for monitoring

### For Skip3DS Authorization Retry

- [ ] Skip3DS result cached during pre-auth
- [ ] Cache checked before authorization retry
- [ ] Context checked before cache fallback
- [ ] Retry only attempted for retriable error codes
- [ ] Not retried more than once (context key check)
- [ ] Network tokenized payments excluded from retry
- [ ] Chargeback protection evaluated before retry
- [ ] Experiment controls retry logic

### For Merchant Risk Profiles

- [ ] High-risk merchant rules applied when experiment = "on"
- [ ] Medium-risk merchant rules applied when experiment = "on"
- [ ] Low-risk merchant rules applied when experiment = "on"
- [ ] Only one risk profile applied per payment
- [ ] Default rules always evaluated
- [ ] Risk profile results included in decision maker

---

## Common Issues

### Issue 1: Skip3DS Not Working for Retry

**Symptom**: Authorization retry always enforces 3DS even when skip3DS evaluation = true

**Root Cause**: Skip3DS result not cached or cache miss during retry

**Fix**:
1. Verify Shield evaluates skip3DS during pre-auth leg
2. Check cache key: `skip3ds_risk_check_{paymentId}`
3. Ensure cache TTL hasn't expired
4. Add fallback to context check before cache

---

### Issue 2: Wrong Risk Profile Applied

**Symptom**: High-risk merchant getting low-risk rules (or vice versa)

**Root Cause**: Splitz experiment not properly configured

**Fix**:
1. Verify Splitz experiment for merchant risk profile
2. Check experiment keys: `EvaluateMerchantWithHighRiskProfile`, etc.
3. Ensure only one risk profile experiment returns "on"

---

### Issue 3: Sardine Decision Ignored

**Symptom**: Sardine allows skip3DS but payment still enforces 3DS

**Root Cause**: Sardine honor experiment is "control"

**Fix**:
1. Check `HonorSardineMerchantDecisionMakerResponse` Splitz experiment
2. Verify experiment returns "on" for test cohort
3. Check Sardine decision maker event logs

---

## Related Documentation

- [Payments-Card Patterns](#payments-card-cross-border-patterns) - Authorization retry and risk evaluation
- [Shield Cross-Border Module](#skip3ds-cross-border-patterns) - Rule engine architecture (if exists)

---

## Code Locations Reference

### Shield

- **Rule Engine Core**: `app/crossBorder/app/services/skip3DS/core.go:44-211`
- **Default Rules**: `app/crossBorder/app/services/skip3DS/default_merchants/`
- **High-Risk Rules**: `app/crossBorder/app/services/skip3DS/high_risk_merchants/`
- **Medium-Risk Rules**: `app/crossBorder/app/services/skip3DS/medium_risk_merchants/`
- **Low-Risk Rules**: `app/crossBorder/app/services/skip3DS/low_risk_merchants/`
- **Sardine Rules**: `app/crossBorder/app/services/skip3DS/sardine_merchants/`
- **Decision Maker**: `app/crossBorder/app/services/skip3DS/decision_maker/`

### Payments-Card

- **Authorization Retry**: `internal/workflow/pay/pxb_helper.go:40-150`
- **Chargeback Protection**: `internal/workflow/pay/pxb_helper.go:110-122`
- **Risk Check**: `internal/workflow/steps/check_transaction_risk.go:37-164`
- **Cross-Border Utils**: `internal/pxb_utils/` (network token check)

---

**Version**: 1.0.0
**Last Updated**: 2026-03-13
**Maintainer**: Cross-Border Code Review Skill


---

# Cross-Border Recurring Payments Patterns

This document explains cross-border specific patterns for recurring/subscription payments across payments-card service.

## Overview

**Service**: payments-card
**Scope**: Recurring payments (subscriptions, auto-debit) for international cards

**Key Differences from One-Time Payments**:
- Recurring payments can be both domestic and international
- Token service integration differs for recurring vs. non-recurring
- Max amount handling for domestic recurring (not for international)
- DCC/MCC NOT supported for recurring payments (forex_applied check)
- Special validation rules for international recurring

---

## Critical Pattern 1: Recurring + International Validation

**Problem**: Can recurring payments be international? What restrictions apply?

### Validation Rules

**Location**: `payments-card/internal/payment/processor/validate.go`

**Rule**: Recurring payments are NOT supported for domestic non-INR payments (forex_applied = true)

```go
// ❌ WRONG - Allowing recurring for domestic forex_applied payments
if payment.GetRecurring() {
    // Process recurring payment without checking forex_applied
}

// ✅ CORRECT - Block recurring for domestic forex_applied
if payment.GetRecurring() && payment.GetForexApplied() &&
   !payment.IsInternational() {
    return errors.New("Recurring not supported for domestic forex payments")
}
```

**Why Critical**:
- Domestic forex_applied payments are non-INR domestic payments (e.g., USD payment by US merchant in India)
- Recurring logic expects INR for domestic payments
- International recurring is allowed but has different handling

**Check Logic**:
```
IF payment.recurring = true
AND forex_applied = true
AND international = false
THEN flag: "Recurring not supported for domestic forex-applied payments"
```

---

## Critical Pattern 2: Token Service Integration for Recurring

**Problem**: When should recurring payments use token service vs. API service for tokenization?

### Token Service Eligibility

**Location**: `payments-card/internal/payment/processor/tokens/token.go:CanCreateTokenInTokenService()` (line 48-87)

**Decision Logic**:

```go
func CanCreateTokenInTokenService(merchant, card, payment) bool {
    // ❌ Block 1: Non-India merchant OR recurring payments return false
    if merchant.GetCountryCode() != "IN" || payment.GetRecurring() {
        return false  // Recurring tokens handled separately via Splitz
    }

    // ✅ Block 2: Cross-border tokenization (non-recurring only)
    if IsCrossBorder(merchant.CountryCode, card.Country) &&
       payment.Analytics.Library == "checkout.js" {
        if card.IsAmex() {
            return false  // Amex excluded
        }
        return internationalTokenApplicable(ctx, merchant, card)  // Splitz check
    }

    // Block 3: Domestic tokenization via Splitz
    return checkSplitzExperiment(merchant, card)
}
```

**Critical Anti-Pattern**:

```go
// ❌ WRONG - Creating token in token service for recurring
if payment.IsInternational() {
    createTokenInTokenService(...)  // Ignores recurring flag!
}

// ✅ CORRECT - Check recurring first
if !payment.GetRecurring() && payment.IsInternational() {
    createTokenInTokenService(...)
}
```

**Why Critical**:
- Recurring tokens have separate Splitz experiment control
- Token service logic differs for recurring vs. non-recurring
- Incorrect tokenization breaks subscription flows

---

## Critical Pattern 3: Max Amount Handling for Recurring

**Problem**: When should max_amount be set in token creation for recurring payments?

### Max Amount Logic

**Location**: `payments-card/internal/token/request/request.go:NewTokenRequest()` (line 130)

**Implementation**:

```go
// Max amount is set ONLY for domestic recurring payments
if payment.GetRecurring() == true && !payment.IsInternational() {
    // Set recurring.max_amount for domestic subscriptions
    request.Recurring = &RecurringData{
        MaxAmount: payment.GetAmount(),
    }
}
```

**Critical Check**:

```go
// ❌ WRONG - Setting max_amount for international recurring
if payment.GetRecurring() {
    request.Recurring = &RecurringData{
        MaxAmount: payment.GetAmount(),  // Wrong for international!
    }
}

// ✅ CORRECT - Only for domestic recurring
if payment.GetRecurring() && !payment.IsInternational() {
    request.Recurring = &RecurringData{
        MaxAmount: payment.GetAmount(),
    }
}
```

**Test Case**: `TestNewTokenRequest_RecurringInternational_MaxAmountNotSet`
- Validates that `request.GetRecurring()` is nil for international payments
- Ensures max_amount block not entered

**Why Critical**:
- International recurring payments have different amount handling
- Setting max_amount for international recurring causes validation failures
- Domestic recurring needs max_amount for mandate enforcement

---

## High-Priority Pattern 1: Recurring Token Count Update

**Problem**: How to track token usage for recurring payments?

### Token Count Management

**Location**: `payments-card/internal/payment/processor/tokens/token.go:UpdateTokenCount()` (line 369)

```go
func UpdateTokenCount(ctx, payment) {
    _, errUpdate := provider.Token.UpdateToken(ctx, payment, nil)
    if errUpdate != nil {
        logger.Error(ctx, "Token update failed")
    }
}
```

**Usage**: Called after successful recurring payment authorization

**Why Important**:
- Tracks how many times a recurring token was used
- Helps identify fraudulent recurring token usage patterns
- Required for token lifecycle management

---

## High-Priority Pattern 2: Recurring Metadata Events

**Problem**: How to track recurring payment metadata for analytics?

### Event Emission

**Location**: `payments-card/internal/constants/event.go` (line 85)

**Constant**: `PaymentInternationalRecurringMetaData = "PAYMENT.INTERNATIONAL_RECURRING.METADATA"`

**Usage**:

```go
// ❌ WRONG - Not emitting recurring metadata event
if payment.IsInternational() && payment.GetRecurring() {
    // Process payment without event
}

// ✅ CORRECT - Emit recurring metadata event
if payment.IsInternational() && payment.GetRecurring() {
    events.Push(ctx, events.PaymentInternationalRecurringMetaData, metadata)
}
```

**Why Important**:
- Analytics team tracks international recurring success rates
- Helps identify recurring payment failure patterns
- Required for business reporting

---

## High-Priority Pattern 3: Recurring Token Webhook

**Problem**: How to update recurring token status after authorization?

### Token Webhook Update

**Location**: `payments-card/internal/payment/processor/tokens/token.go:UpdateTokenOnAuthorized()` (line 489)

```go
func UpdateTokenOnAuthorized(ctx, tokenId, merchantId, customerId, recurringStatus, recurringFailureReason) {
    data := map[string]interface{}{
        "token_id":                 tokenId,
        "merchant_id":              merchantId,
        "customer_id":              customerId,
        "recurring_status":         recurringStatus,
        "recurring_failure_reason": recurringFailureReason,
    }

    tResponse, err := provider.Token.SendWebhook(ctx, data)
    if err != nil {
        logger.Error(ctx, "Token webhook failed")
    }
}
```

**Parameters**:
- `recurringStatus`: "confirmed", "rejected", "paused"
- `recurringFailureReason`: Error code if recurring authorization failed

**Why Important**:
- Updates token service with recurring authorization result
- Triggers webhook to merchant about recurring status
- Required for subscription lifecycle management

---

## High-Priority Pattern 4: Forex Applied Check for Recurring

**Problem**: Should DCC/MCC be applied to recurring payments?

### Forex Applied Validation

**Business Rule**: **Recurring payments do NOT support DCC/MCC**

```go
// ❌ WRONG - Applying DCC to recurring payment
if payment.IsInternational() {
    applyDCC(payment)  // Wrong for recurring!
}

// ✅ CORRECT - Check recurring before DCC
if payment.IsInternational() && !payment.GetRecurring() {
    applyDCC(payment)
}
```

**Validation**:

```go
// ❌ WRONG - Allowing forex_applied = true for recurring
if payment.GetRecurring() && payment.GetForexApplied() {
    // Process recurring with forex - incorrect!
}

// ✅ CORRECT - Block or handle appropriately
if payment.GetRecurring() && payment.GetForexApplied() {
    if !payment.IsInternational() {
        return errors.New("Recurring not supported for domestic forex payments")
    }
    // International recurring: forex_applied should be false or handle separately
}
```

**Why Important**:
- Recurring payments use fixed rates, not dynamic DCC rates
- DCC requires customer interaction, not compatible with auto-debit
- Forex charge calculation differs for recurring

---

## Detection Patterns for Code Review

### Pattern 1: Recurring + Domestic Forex Applied

**🚨 CRITICAL**

```go
// ❌ WRONG - Missing validation
if payment.GetRecurring() {
    processPayment(payment)  // No forex_applied check
}

// ✅ CORRECT - Validate forex_applied
if payment.GetRecurring() {
    if payment.GetForexApplied() && !payment.IsInternational() {
        return errors.New("Recurring not supported for domestic forex payments")
    }
    processPayment(payment)
}
```

**Where to Check**:
- Payment validation logic
- Recurring payment processing
- Forex charge creation

---

### Pattern 2: Token Service Creation for Recurring

**⚠️ HIGH-PRIORITY**

```go
// ❌ WRONG - Creating token in token service for recurring
if CanCreateTokenInTokenService(merchant, card, payment) {
    createToken(...)  // Might be recurring payment!
}

// ✅ CORRECT - Explicitly check recurring
if !payment.GetRecurring() && CanCreateTokenInTokenService(...) {
    createToken(...)
}
// OR: Let CanCreateTokenInTokenService handle it (returns false for recurring)
```

**Where to Check**:
- Token creation logic
- Payment authorization callbacks
- Tokenization decision points

---

### Pattern 3: Max Amount for International Recurring

**⚠️ HIGH-PRIORITY**

```go
// ❌ WRONG - Setting max_amount for international recurring
if payment.GetRecurring() {
    request.Recurring = &RecurringData{
        MaxAmount: payment.GetAmount(),  // Wrong!
    }
}

// ✅ CORRECT - Only for domestic recurring
if payment.GetRecurring() && !payment.IsInternational() {
    request.Recurring = &RecurringData{
        MaxAmount: payment.GetAmount(),
    }
}
```

**Where to Check**:
- Token request building
- Recurring payment setup
- Subscription creation flows

---

### Pattern 4: DCC/MCC for Recurring Payments

**🚨 CRITICAL**

```go
// ❌ WRONG - Applying DCC to recurring
if payment.IsInternational() {
    dccResponse := crossBorder.ForexChargesInternalForDCC(...)  // Wrong!
}

// ✅ CORRECT - Check recurring first
if payment.IsInternational() && !payment.GetRecurring() {
    dccResponse := crossBorder.ForexChargesInternalForDCC(...)
}
```

**Where to Check**:
- DCC application logic
- Forex charge creation
- Cross-border payment processing

---

## Testing Checklist

### For Cross-Border Recurring Payments

- [ ] Recurring payment with international = true processes correctly
- [ ] Recurring payment with forex_applied = true AND international = false is blocked
- [ ] Max amount NOT set for international recurring payments
- [ ] Max amount IS set for domestic recurring payments
- [ ] Token service returns false for recurring payment tokenization
- [ ] Recurring token count updated after successful authorization
- [ ] Recurring metadata events emitted for international recurring
- [ ] Token webhook sent with recurring status after authorization

### For Recurring + Forex

- [ ] DCC not offered for recurring payments
- [ ] MCC not applied to recurring payments
- [ ] Forex charge creation skipped for recurring
- [ ] forex_applied flag false or properly handled for recurring

### For Recurring Token Management

- [ ] Token created via appropriate service (token-service vs API)
- [ ] Recurring splitz experiment checked before token creation
- [ ] Token count incremented on recurring payment
- [ ] Token deletion removes recurring token properly

---

## Common Issues

### Issue 1: Recurring Payment Fails with "Forex Applied" Error

**Symptom**: Domestic non-INR recurring payment rejected

**Root Cause**: forex_applied = true for domestic payment

**Fix**:
1. Check if payment is domestic (merchant country = card country = non-India)
2. Verify forex_applied flag is false
3. If forex_applied = true, ensure validation blocks recurring

---

### Issue 2: International Recurring Token Has Max Amount

**Symptom**: Token creation fails for international recurring with max_amount

**Root Cause**: Max amount set for international recurring payment

**Fix**:
1. Check `NewTokenRequest()` logic
2. Ensure `!payment.IsInternational()` condition before setting max_amount
3. Verify test case `TestNewTokenRequest_RecurringInternational_MaxAmountNotSet`

---

### Issue 3: Recurring Token Created in Wrong Service

**Symptom**: Token service error or fallback to API for recurring

**Root Cause**: `CanCreateTokenInTokenService()` returns true for recurring

**Fix**:
1. Verify `payment.GetRecurring()` check returns false
2. Check Splitz experiment for recurring tokens
3. Ensure separate token creation path for recurring

---

## Metrics & Monitoring

### Prometheus Metrics

**Recurring Labels** (from `pkg/monitoring/prometheus/metric_config.go`):
- `recurring`: "true"/"false"
- `recurring_type`: "initial"/"auto"/"final"

**Usage**:

```go
prometheus.InstrumentPaymentCreated(
    merchant.CountryCode,
    payment.International,
    payment.Recurring,      // "true" for recurring
    payment.RecurringType,  // "initial", "auto", "final"
)
```

**Why Important**:
- Track recurring payment success rates separately
- Monitor international vs. domestic recurring
- Identify recurring type-specific issues

---

## Related Documentation

- [Payments-Card Patterns](#payments-card-cross-border-patterns) - General DCC and CFB patterns
- [Token Service Documentation](../../tokens/README.md) - Token management (if available)

---

## Code Locations Reference

### Payments-Card

- **Recurring Validation**: `internal/payment/processor/validate.go`
- **Token Service Check**: `internal/payment/processor/tokens/token.go:48-87`
- **Max Amount Logic**: `internal/token/request/request.go:130`
- **Token Count Update**: `internal/payment/processor/tokens/token.go:369-375`
- **Token Webhook**: `internal/payment/processor/tokens/token.go:489-507`
- **Recurring Events**: `internal/constants/event.go:85`
- **Recurring Metrics**: `pkg/monitoring/prometheus/metric_config.go:44-45`

### Test Coverage

- **Recurring International Test**: `internal/token/tokens_test.go:1608`
- **Validate Test**: `internal/payment/processor/validation_test.go`
- **SLIT Tests**: `slit/test_suites/mandate_recurring_test.go`

---

**Version**: 1.0.0
**Last Updated**: 2026-03-13
**Maintainer**: Cross-Border Code Review Skill


---

# Network Tokenization Cross-Border Patterns

This document explains network tokenization patterns for international card payments across payments-card, vault, and token-service.

## Overview

**Services Involved**:
- `payments-card` - Network token processing and routing decisions
- `vault` - Token storage and retrieval (network tokens from card networks)
- `token-service` (tokens) - Razorpay tokenization service

**What is Network Tokenization**:
- Card networks (Visa, Mastercard) provide tokens to replace actual card numbers
- Identified by **Token IIN** (different from actual card IIN)
- Used for secure recurring/saved card payments
- Cross-border network tokens have special handling

---

## Critical Pattern 1: Cross-Border Network Token Detection

**Problem**: How to identify if a payment is using a cross-border network token?

### IsCBNetworkTokenised Check

**Location**: `payments-card/internal/pxb_utils/utils.go:IsCBNetworkTokenised()` (line 66-75)

**Implementation**:

```go
func IsCBNetworkTokenised(merchantCountryCode, cardCountry string, isTokenised bool, tokenIIN string) bool {
    // Step 1: Check if cross-border
    if !IsCrossBorder(merchantCountryCode, cardCountry) {
        return false  // Not cross-border
    }

    // Step 2: Check if tokenised
    if !isTokenised {
        return false  // Not tokenised
    }

    // Step 3: Check if Token IIN is present
    return !utils.IsEmpty(strings.TrimSpace(tokenIIN))
}

func IsCrossBorder(merchantCountryCode, cardCountryCode string) bool {
    merchantCountry := strings.ToUpper(strings.TrimSpace(merchantCountryCode))
    cardCountry := strings.ToUpper(strings.TrimSpace(cardCountryCode))

    if utils.IsEmpty(merchantCountry) || utils.IsEmpty(cardCountry) {
        return false
    }

    // Cross-border: Indian merchant & non-Indian card
    return merchantCountry == "IN" && cardCountry != "IN"
}
```

**Critical Check**:

```go
// ❌ WRONG - Only checking tokenised flag
if card.Tokenised {
    processAsNetworkToken(...)  // Missing IIN and cross-border check!
}

// ✅ CORRECT - Check all three conditions
if IsCBNetworkTokenised(merchant.CountryCode, card.Country, card.Tokenised, card.TokenIIN) {
    processAsNetworkToken(...)
}
```

**Why Critical**:
- Network tokens require special routing logic
- Skip3DS retry logic MUST exclude network tokens
- Smart retry logic blocks network tokens
- Wrong detection causes authorization failures

---

## Critical Pattern 2: Network Token Routing Decision

**Problem**: When should a saved card payment use network token vs. plain card number?

### Routing Logic

**Location**: `payments-card/internal/payment/processor/tokens/token.go:shouldRouteViaNetworkToken()` (line 509-527)

**Decision Tree**:

```go
func shouldRouteViaNetworkToken(ctx, merchantId string, cvvEmpty bool) bool {
    // Rule 1: If CVV is empty, ALWAYS route via network token
    if cvvEmpty {
        return true  // No CVV → use network token
    }

    // Rule 2: CVV is present → check Splitz experiment
    splitzResp := splitz.GetClient().GetVariant(ctx,
        splitzReq.GetEvaluateRequestByName(ctx, uuid.NewString(),
            splitz.CBNetworkTokenRoute,
            map[string]interface{}{
                "merchant_id": merchantId,
            },
        ),
    )

    enabled := splitzResp != nil && splitzResp.Name == "variant_on"
    return enabled  // Experiment controls CVV-present routing
}
```

**Usage in Token Fetch**:

**Location**: `payments-card/internal/payment/processor/tokens/token.go:FetchTokenFromTokenService()` (line 181-198)

```go
// After fetching token from token service
if IsCrossBorder(countryCode, cardCountry) &&
   !utils.IsEmpty(responseToken.Data[0].Card.TokenIin) &&
   responseToken.Data[0].MerchantId == payment.GetMerchantId() &&
   cardNetwork != "amex" {  // Amex excluded

    cvvEmpty := utils.IsEmpty(strings.TrimSpace(input.Card.Cvv))

    // ✅ Critical decision point
    if shouldRouteViaNetworkToken(ctx, payment.GetMerchantId(), cvvEmpty) {
        input.Card.TokenIIN = responseToken.Data[0].Card.TokenIin  // Use network token
        input.Card.International = true
        input.Card.Tokenized = true
    }

    // Push event for analytics
    pushCBNetworkTokenEvent(ctx, merchantId, paymentId, cardCountry, cardNetwork, input.Card, cvvEmpty)
}
```

**Critical Anti-Pattern**:

```go
// ❌ WRONG - Always using network token if available
if !utils.IsEmpty(token.TokenIIN) {
    input.Card.TokenIIN = token.TokenIIN  // Ignores CVV and experiment!
}

// ✅ CORRECT - Check routing decision
if shouldRouteViaNetworkToken(ctx, merchantId, cvvEmpty) {
    input.Card.TokenIIN = token.TokenIIN
}
```

**Why Critical**:
- CVV-less transactions MUST use network token for security
- CVV-present transactions can use either based on experiment
- Incorrect routing causes payment failures or security issues

---

## Critical Pattern 3: Network Token Exclusion from Smart Retry

**Problem**: Should cross-border network tokenized payments be eligible for smart retry?

### Smart Retry Eligibility

**Location**: `payments-card/internal/pxb_utils/utils.go:IsCBSmartRetryEligible()` (line 77-94)

**Implementation**:

```go
func IsCBSmartRetryEligible(params CBSmartRetryParams) bool {
    // Block 1: Non-Razorpay org merchants
    if params.MerchantOrgID != constants.RazorpayOrgID {
        return false
    }

    // Block 2: Check terminal gateway (only Fulcrum or Hitachi)
    if !utils.IsEmpty(params.Gateway) &&
       params.Gateway != constants.Fulcrum &&
       params.Gateway != constants.Hitachi {
        return false
    }

    // ✅ Block 3: CRITICAL - Exclude CB network tokenised payments
    if IsCBNetworkTokenised(params.MerchantCountryCode, params.CardCountry,
                            params.IsTokenised, params.TokenIIN) {
        return false  // Network tokens cannot use smart retry
    }

    return true
}
```

**Critical Check**:

```go
// ❌ WRONG - Smart retry without network token check
if authorizationFailed && payment.IsInternational() {
    attemptSmartRetry(payment)  // Might be network token!
}

// ✅ CORRECT - Check network token exclusion
params := CBSmartRetryParams{
    MerchantCountryCode: merchant.CountryCode,
    CardCountry:        card.Country,
    IsTokenised:        card.Tokenised,
    TokenIIN:          card.TokenIIN,
}
if authorizationFailed && IsCBSmartRetryEligible(params) {
    attemptSmartRetry(payment)
}
```

**Why Critical**:
- Network tokens have different routing requirements
- Smart retry may route to incompatible gateways
- Authorization failures with network tokens need different handling
- Skip3DS retry also excludes network tokens (pattern in skip3ds-patterns.md)

---

## Critical Pattern 4: Amex Exclusion from Network Tokenization

**Problem**: American Express cards have different tokenization handling.

### Amex Exclusion Logic

**Location**: `payments-card/internal/payment/processor/tokens/token.go:CanCreateTokenInTokenService()` (line 56-62)

```go
// Cross-border tokenisation (via StandardCheckout only)
if IsCrossBorder(merchant.CountryCode, card.Country) &&
   payment.Analytics.Library == "checkout.js" {

    // ✅ CRITICAL - Amex excluded from network tokenization
    if card.IsAmex() {
        return false  // No token service for Amex
    }

    return internationalTokenApplicable(ctx, merchant, card)
}
```

**Token Fetch Logic** (line 184):

```go
if IsCrossBorder(countryCode, cardCountry) &&
   !utils.IsEmpty(responseToken.Card.TokenIin) &&
   responseToken.MerchantId == payment.MerchantId &&
   cardNetwork != "amex" {  // ✅ Exclude Amex

    // Proceed with network token routing decision
}
```

**Critical Check**:

```go
// ❌ WRONG - Including Amex in network tokenization
if IsCrossBorder(merchant, card) {
    createNetworkToken(card)  // Amex not excluded!
}

// ✅ CORRECT - Exclude Amex
if IsCrossBorder(merchant, card) && !card.IsAmex() {
    createNetworkToken(card)
}
```

**Why Critical**:
- Amex has different tokenization agreements
- Amex network tokens may not be supported by all gateways
- Including Amex causes tokenization failures

---

## High-Priority Pattern 1: Token IIN vs. Card IIN

**Problem**: How to differentiate between token IIN and actual card IIN?

### IIN Handling

**Location**: `payments-card/internal/payment/processor/tokens/token.go:FetchTokenFromTokenServiceAndPersist()` (line 254-258)

```go
// Preference for Token IIN
if !utils.IsEmpty(responseToken.Card.TokenIin) {
    fetchTokenCardEntityResponse.TokenIIN = responseToken.Card.TokenIin  // ✅ Prefer Token IIN
} else {
    fetchTokenCardEntityResponse.TokenIIN = responseToken.Card.Iin  // Fallback to card IIN
}
```

**Why Important**:
- Token IIN identifies the tokenization provider
- Card IIN is the original card's BIN
- Gateways need Token IIN for network token processing
- Sending wrong IIN causes authorization failures

---

## High-Priority Pattern 2: CVV Handling for Network Tokens

**Problem**: What CVV value to send for network tokenized payments?

### CVV Logic

**Location**: `payments-card/internal/payment/processor/tokens/token.go` (line 200-202, 484-486)

**For Visa Network Tokens without CVV**:

```go
if utils.IsEmpty(input.Card.Cvv) && responseToken.Card.GetNetwork() == "Visa" {
    input.Card.Cvv = "123"  // ✅ Dummy CVV for Visa network tokens
}
```

**For Issuer/Dual Tokens**:

```go
func createDummyCardForIssuerOrDualToken(input, responseToken) {
    // ... other fields
    if utils.IsEmpty(input.Card.Cvv) {
        input.Card.Cvv = constants.DummyCardCVV  // ✅ Dummy CVV
    }
}
```

**Why Important**:
- Some gateways require CVV field even for network tokens
- Empty CVV causes validation errors
- Dummy CVV (e.g., "123") satisfies validation without actual CVV

---

## High-Priority Pattern 3: Network Token Event Tracking

**Problem**: How to track network token usage for analytics?

### Event Emission

**Location**: `payments-card/internal/payment/processor/tokens/token.go:pushCBNetworkTokenEvent()` (line 529-550)

```go
func pushCBNetworkTokenEvent(ctx, merchantId, paymentId, cardCountry, cardNetwork string, card request.Card, cvvEmpty bool) {
    tokenIINSet := !utils.IsEmpty(card.TokenIIN)

    // Determine card type
    cardType := constants.CBCardTypeRZPSavedCard
    if tokenIINSet {
        if !cvvEmpty {
            cardType = constants.CBCardTypeNetworkTokenWithCVV
        } else {
            cardType = constants.CBCardTypeNetworkTokenSkippedCVV
        }
    }

    eventPayload := map[string]interface{}{
        "merchant_id":  merchantId,
        "payment_id":   paymentId,
        "card_country": cardCountry,
        "card_network": cardNetwork,
        "card_type":    cardType,
    }

    events.PushPaymentCreateEvents(ctx, events.CBNetworkTokenProcessed, &eventPayload, nil)
}
```

**Card Types**:
- `CBCardTypeRZPSavedCard`: Razorpay saved card (no network token)
- `CBCardTypeNetworkTokenWithCVV`: Network token with CVV provided
- `CBCardTypeNetworkTokenSkippedCVV`: Network token without CVV (CVV-less flow)

**Why Important**:
- Analytics team tracks network token adoption
- Helps identify CVV-less vs. CVV-present flows
- Required for business reporting and optimization

---

## High-Priority Pattern 4: Vault Integration for Network Tokens

**Problem**: How to fetch network token details from vault for special cases?

### Vault Token Fetch

**Location**: `payments-card/internal/payment/processor/tokens/token.go:getAdditionalInputForDinersTokenizedPayment()` (line 424-451)

**Use Case**: Diners (HDFC) tokenized payments need special parameters

```go
func getAdditionalInputForDinersTokenizedPayment(ctx, fetchTokenCardEntityResponse, paymentId, merchantId, vaultToken) {
    // ✅ Fetch network token from vault
    networkTokens, err := provider.Vault.FetchTokens(ctx, paymentId, vault.FetchTokenRequest{
        Token:    vaultToken,
        Merchant: vault.MerchantRequest{MerchantId: merchantId},
    })
    if err != nil || !networkTokens.Success {
        logger.Error(ctx, "Vault token fetch error")
        return err
    }

    // Extract token_reference_number from vault
    fetchTokenCardEntityResponse.TokenReferenceNumber =
        networkTokens.ServiceProviders[0].ProviderData.TokenReferenceNumber

    // Fetch tokenization terminal for token_requestor_id
    terminal, err := provider.Terminals.GetTerminalByID(ctx,
        networkTokens.ServiceProviders[0].TokenisedTerminalId)
    if err != nil {
        return err
    }

    fetchTokenCardEntityResponse.TokenReferenceId = terminal.GatewayMerchantID

    logger.Info(ctx, "Diners tokenized payment additional parameters",
        "token_reference_number", fetchTokenCardEntityResponse.TokenReferenceNumber,
        "token_reference_id", fetchTokenCardEntityResponse.TokenReferenceId)

    return nil
}
```

**Why Important**:
- Issuer tokens (Providers, Axis, HDFC) need additional parameters
- Token reference number identifies the specific token instance
- Token requestor ID links to tokenization terminal
- Missing parameters cause Diners payment failures

---

## Detection Patterns for Code Review

### Pattern 1: Missing Network Token Check

**🚨 CRITICAL**

```go
// ❌ WRONG - Processing tokenized payment without network token check
if card.Tokenised {
    processPayment(card)  // Missing IsCBNetworkTokenised check!
}

// ✅ CORRECT - Check if cross-border network token
if IsCBNetworkTokenised(merchant.CountryCode, card.Country, card.Tokenised, card.TokenIIN) {
    processNetworkTokenPayment(card)
} else if card.Tokenised {
    processRazorpayTokenPayment(card)
}
```

**Where to Check**:
- Payment processing logic
- Authorization flow
- Smart retry eligibility
- Skip3DS retry logic

---

### Pattern 2: Incorrect Token Routing Decision

**⚠️ HIGH-PRIORITY**

```go
// ❌ WRONG - Always using network token if Token IIN exists
if !utils.IsEmpty(token.TokenIIN) {
    input.Card.TokenIIN = token.TokenIIN  // Missing routing decision!
}

// ✅ CORRECT - Check routing decision
cvvEmpty := utils.IsEmpty(input.Card.Cvv)
if shouldRouteViaNetworkToken(ctx, merchantId, cvvEmpty) {
    input.Card.TokenIIN = token.TokenIIN
}
```

**Where to Check**:
- Token fetch from token service
- Saved card payment processing
- Cross-border payment flows

---

### Pattern 3: Amex Included in Network Tokenization

**🚨 CRITICAL**

```go
// ❌ WRONG - Creating network token for Amex
if IsCrossBorder(merchant, card) {
    createNetworkToken(card)  // Amex not excluded!
}

// ✅ CORRECT - Exclude Amex
if IsCrossBorder(merchant, card) && card.Network != "amex" {
    createNetworkToken(card)
}
```

**Where to Check**:
- Token creation logic
- Cross-border tokenization
- Network token routing

---

### Pattern 4: Network Token Included in Smart Retry

**🚨 CRITICAL**

```go
// ❌ WRONG - Smart retry without checking network token
if authFailed && payment.IsInternational() {
    smartRetry(payment)  // Might be network token!
}

// ✅ CORRECT - Exclude network tokens
params := CBSmartRetryParams{
    MerchantCountryCode: merchant.CountryCode,
    CardCountry:        card.Country,
    IsTokenised:        card.Tokenised,
    TokenIIN:          card.TokenIIN,
}
if authFailed && IsCBSmartRetryEligible(params) {
    smartRetry(payment)
}
```

**Where to Check**:
- Smart retry logic
- Authorization retry flows
- Cross-border payment recovery

---

## Testing Checklist

### For Network Tokenization Detection

- [ ] `IsCBNetworkTokenised` returns true for: Indian merchant + foreign card + tokenised + Token IIN present
- [ ] `IsCBNetworkTokenised` returns false if any condition missing
- [ ] Amex excluded from network tokenization
- [ ] Token IIN preferred over card IIN when available

### For Routing Decisions

- [ ] CVV-empty network token payments always routed via network token
- [ ] CVV-present network token payments follow Splitz experiment
- [ ] Splitz experiment `CBNetworkTokenRoute` controls CVV-present routing
- [ ] Event emitted with correct card type (with CVV vs without CVV)

### For Smart Retry Exclusion

- [ ] Network tokens excluded from smart retry eligibility
- [ ] Skip3DS retry logic excludes network tokens
- [ ] Smart retry only for Razorpay org merchants
- [ ] Smart retry only for Fulcrum/Hitachi gateways

### For Vault Integration

- [ ] Network token fetched from vault for Diners payments
- [ ] Token reference number extracted correctly
- [ ] Token requestor ID from tokenization terminal
- [ ] Vault fetch errors handled gracefully

---

## Common Issues

### Issue 1: Network Token Payment Fails with "Invalid IIN"

**Symptom**: Gateway rejects payment with invalid IIN error

**Root Cause**: Sending card IIN instead of Token IIN

**Fix**:
1. Verify Token IIN is populated in `input.Card.TokenIIN`
2. Check preference logic: Token IIN > Card IIN
3. Ensure `shouldRouteViaNetworkToken` returns true

---

### Issue 2: Smart Retry Fails for Network Token

**Symptom**: Smart retry attempted for network token payment, fails

**Root Cause**: `IsCBSmartRetryEligible` not excluding network tokens

**Fix**:
1. Verify `IsCBNetworkTokenised` check in smart retry logic
2. Ensure Token IIN passed to eligibility check
3. Confirm network token returns false from eligibility

---

### Issue 3: Amex Network Tokenization Error

**Symptom**: Amex tokenization fails or returns error

**Root Cause**: Amex not excluded from network tokenization

**Fix**:
1. Add Amex exclusion in `CanCreateTokenInTokenService`
2. Check network != "amex" before tokenization
3. Verify Amex uses fallback flow

---

### Issue 4: CVV-less Network Token Validation Error

**Symptom**: Gateway rejects CVV-less network token with validation error

**Root Cause**: CVV field empty for network token

**Fix**:
1. Check dummy CVV assignment for Visa network tokens
2. Ensure "123" or `DummyCardCVV` set when CVV empty
3. Verify gateway accepts dummy CVV for network tokens

---

## Metrics & Monitoring

### Event Tracking

**Event**: `events.CBNetworkTokenProcessed`

**Payload**:
```json
{
  "merchant_id": "string",
  "payment_id": "string",
  "card_country": "US",
  "card_network": "visa",
  "card_type": "network_token_skipped_cvv"  // or "network_token_with_cvv" or "rzp_saved_card"
}
```

**Use Cases**:
- Track network token adoption rate
- Monitor CVV-less vs CVV-present split
- Identify network-specific patterns

---

## Related Documentation

- [Skip3DS Patterns](#skip3ds-cross-border-patterns) - Skip3DS exclusion for network tokens
- [Wallet Payments Patterns](#wallet-payments-apple-pay--google-pay-cross-border-patterns) - Apple Pay/Google Pay tokenization
- [Vault Service Documentation](../../vault/README.md) - Token storage (if available)
- [Token Service Documentation](../../tokens/README.md) - Razorpay tokenization (if available)

---

## Code Locations Reference

### Payments-Card

- **Network Token Detection**: `internal/pxb_utils/utils.go:66-75`
- **Routing Decision**: `internal/payment/processor/tokens/token.go:509-527`
- **Token Fetch Logic**: `internal/payment/processor/tokens/token.go:181-204`
- **Smart Retry Eligibility**: `internal/pxb_utils/utils.go:77-94`
- **Amex Exclusion**: `internal/payment/processor/tokens/token.go:56-62, 184`
- **CVV Handling**: `internal/payment/processor/tokens/token.go:200-202, 484-486`
- **Event Emission**: `internal/payment/processor/tokens/token.go:529-550`
- **Vault Integration**: `internal/payment/processor/tokens/token.go:424-451`
- **IIN Preference**: `internal/payment/processor/tokens/token.go:254-258`

### Test Coverage

- **CB Network Token Tests**: `slit/test_suites/cb_network_token_test.go`
- **Tokenised Payment Tests**: `slit/test_suites/tokenised_payment_test.go`
- **Token Service Tests**: `internal/payment/processor/tokens/token_test.go`

---

**Version**: 1.0.0
**Last Updated**: 2026-03-13
**Maintainer**: Cross-Border Code Review Skill


---

# Common Anti-Patterns

This document compiles frequently encountered mistakes in cross-border payment code. Each anti-pattern includes the wrong code, correct code, impact, and real-world example.

## Anti-Pattern 1: Fee Not Subtracted Before Markdown

### Wrong Code

```go
// CFB fee handling - WRONG
func calculateBaseAmount(amount, fee int64, rate, markdown float64) int64 {
    markdownRate := rate - rate * markdown / 100
    baseAmount := math.Ceil(markdownRate * amount / 100)
    baseFee := math.Ceil(rate * fee / 100)
    return baseAmount + baseFee  // Fee not subtracted from amount!
}
```

### Correct Code

```go
// CFB fee handling - CORRECT
func calculateBaseAmount(amount, fee int64, rate, markdown float64) int64 {
    amount -= fee  // MUST subtract fee first
    markdownRate := rate - rate * markdown / 100
    baseAmount := math.Ceil(markdownRate * amount / 100)
    baseFee := math.Ceil(rate * fee / 100)
    return baseAmount + baseFee
}
```

### Impact

- Merchant receives higher settlement than expected (overpayment)
- Markdown benefit incorrectly applied to fee
- At scale: significant revenue leakage
- **Severity**: 🚨 Critical

### Real Example

Production incident where merchant received $0.61 extra per $100 transaction due to this error.

---

## Anti-Pattern 2: Fee Converted at Markdown Rate

### Wrong Code

```go
// Fee conversion - WRONG
baseFee := math.Ceil(markdownExchangeRate * fee * denominationFactor)
// Using markdown rate for fee!
```

### Correct Code

```go
// Fee conversion - CORRECT
baseFee := math.Ceil(exchangeRate * fee * denominationFactor)
// Using original rate for fee
```

### Impact

- Customer sees incorrect fee amount
- Fee transparency violated
- Merchant receives less fee revenue
- **Severity**: 🚨 Critical

---

## Anti-Pattern 3: Currency Mismatch in Arithmetic

### Wrong Code

```php
// Mixed currency comparison - WRONG
if ($payment->getAmount() - $payment->getFee() < $order->getAmount()) {
    // $payment->amount in USD, $payment->fee in INR - nonsense comparison!
    throw new ValidationException();
}
```

### Correct Code

```php
// Convert to same currency first - CORRECT
$feeInPaymentCurrency = $this->convertCurrency(
    $payment->getFee(),
    Currency::INR,
    $payment->getCurrency()
);

if ($payment->getAmount() - $feeInPaymentCurrency < $order->getAmount()) {
    throw new ValidationException();
}
```

### Impact

- Validation failures for cross-border CFB payments
- Customer unable to complete payment
- Production incident (real case from brainstorm)
- **Severity**: 🚨 Critical

### Real Example

Team deployed PR comparing `$100 USD - ₹50 INR` which caused validation to fail for all cross-border CFB payments.

---

## Anti-Pattern 4: Missing Fee Recalculation

### Wrong Code

```go
// Update with fee - WRONG
func UpdateForexCharges(entity *ForexCharges, req *UpdateRequest) {
    if req.Fee != nil {
        entity.SetFee(req.Fee)  // Only updating fee field
    }
    // baseAmount still calculated without fee!
}
```

### Correct Code

```go
// Update with fee - CORRECT
func UpdateForexCharges(entity *ForexCharges, req *UpdateRequest) {
    if !utils.IsEmpty(req.Fee) {
        baseAmount, baseFee := calculateBaseAmountAndFee(...)
        entity.SetBaseAmount(baseAmount)  // Recalculate!
        entity.SetBaseFee(baseFee)
    }
}
```

### Impact

- Cached baseAmount used with new fee
- Merchant settlement incorrect
- **Severity**: 🚨 Critical

---

## Anti-Pattern 5: Fee Not Converted at Capture

### Wrong Code

```php
// Capture - WRONG
public function capture($payment) {
    // Fee stays in payment currency (USD)
    $txn->setFee($payment->getFee());  // Wrong currency!
    $txn->setCurrency(Currency::INR);
}
```

### Correct Code

```php
// Capture - CORRECT
public function capture($payment) {
    if ($payment->getCurrency() !== Currency::INR
        and $payment->isFeeBearerCustomer()
        and $this->isMCCAppliedPayment($payment)) {
        // Fee from txn is in INR
        $payment->setFee($txn->getFee());
    }
}
```

### Impact

- Merchant dashboard shows fee in wrong currency
- Settlement reports incorrect
- **Severity**: ⚠️ High-Priority

---

## Anti-Pattern 6: Missing Denomination Factor

### Wrong Code

```go
// Conversion - WRONG
convertedAmount := math.Ceil(exchangeRate * amount)
// Missing denominationFactor!
```

### Correct Code

```go
// Conversion - CORRECT
denominationFactor := float64(conversionDenomination) / float64(baseDenomination)
convertedAmount := math.Ceil(exchangeRate * amount * denominationFactor)
```

### Impact

- Wrong conversion amounts (off by 100x or more)
- Especially critical for JPY and other non-decimal currencies
- **Severity**: 🚨 Critical

---

## Anti-Pattern 7: Floor or Round Instead of Ceiling

### Wrong Code

```go
// Rounding - WRONG
baseAmount := math.Floor(markdownRate * amount)  // Floor loses money
// or
baseAmount := math.Round(markdownRate * amount)  // Unpredictable
```

### Correct Code

```go
// Rounding - CORRECT
baseAmount := math.Ceil(markdownRate * amount)  // Always round up
```

### Impact

- Merchant loses money on each transaction
- Accumulates over time
- **Severity**: ⚠️ High-Priority

---

## Anti-Pattern 8: Three-Decimal Currency Not Rounded

### Wrong Code

```go
// KWD/OMR/BHD - WRONG
amount := int64(1237)  // 1.237 KWD
// Send to network without rounding - will fail!
return amount
```

### Correct Code

```go
// KWD/OMR/BHD - CORRECT
func roundOffIfApplicable(amount int64, currencyExponent int64) int64 {
    if currencyExponent == 3 {
        return int64(math.Ceil(float64(amount)*0.1) / 0.1)
    }
    return amount
}
```

### Impact

- Card network rejects transaction
- Payment fails
- **Severity**: 🚨 Critical

---

## Anti-Pattern 9: Division by Zero Not Protected

### Wrong Code

```go
// Division by zero - WRONG
denominationFactor := float64(conversionDenom) / float64(baseDenom)
// Crashes if baseDenom or conversionDenom is 0!
```

### Correct Code

```go
// Division by zero - CORRECT
if baseDenom == 0 || conversionDenom == 0 {
    return 0  // Or return error
}
denominationFactor := float64(conversionDenom) / float64(baseDenom)
```

### Impact

- Service crash
- Payment processing down
- **Severity**: 🚨 Critical

---

## Anti-Pattern 10: Base Currency from Request

### Wrong Code

```go
// Base currency - WRONG
baseCurrency := req.BaseCurrency  // Trusting user input!
```

### Correct Code

```go
// Base currency - CORRECT
func getMerchantBaseCurrency(merchantCountry string) string {
    if merchantCountry == "MY" {
        return currency.MYR
    }
    if merchantCountry == "SG" {
        return currency.SGD
    }
    return currency.INR  // Default
}

baseCurrency := getMerchantBaseCurrency(req.MerchantCountry)
```

### Impact

- Regulatory compliance violation
- Merchant receives wrong currency
- **Severity**: 🚨 Critical

---

## Anti-Pattern 11: INR-INR Not Handled

### Wrong Code

```go
// INR-INR - WRONG
baseAmount := calculateWithMarkdown(amount, fee, rate, markdown)
// Applying conversion even for INR → INR!
```

### Correct Code

```go
// INR-INR - CORRECT
if baseCurrency == paymentCurrency && baseCurrency == "INR" {
    // No conversion needed
    baseAmount = amount
} else {
    baseAmount = calculateWithMarkdown(amount, fee, rate, markdown)
}
```

### Impact

- Unnecessary conversion for domestic payments
- Slight amount differences due to rounding
- **Severity**: ⚠️ High-Priority

---

## Anti-Pattern 12: Silent Zero on Invalid Currency

### Wrong Code

```go
// Invalid currency - WRONG
func convertCurrency(amount, fromCurrency, toCurrency string) int64 {
    if !isValidCurrency(fromCurrency) || !isValidCurrency(toCurrency) {
        return 0  // Silent failure!
    }
    // ... conversion
}
```

### Correct Code

```go
// Invalid currency - CORRECT
func convertCurrency(amount, fromCurrency, toCurrency string) (int64, error) {
    if !isValidCurrency(fromCurrency) || !isValidCurrency(toCurrency) {
        return 0, errors.New("invalid currency code")
    }
    // ... conversion
}
```

### Impact

- Bugs masked (returns 0 instead of error)
- Hard to debug
- **Severity**: ⚠️ High-Priority

---

## Quick Reference Table

| Anti-Pattern | Severity | Impact | Detection |
|--------------|----------|--------|-----------|
| Fee not subtracted before markdown | 🚨 Critical | Merchant overpayment | Check for `amount -= fee` before markdown calc |
| Fee at markdown rate | 🚨 Critical | Incorrect fee display | Verify baseFee uses exchangeRate, not markdownRate |
| Currency mismatch in arithmetic | 🚨 Critical | Validation failures | Check currencies match before +/- operations |
| Missing fee recalculation | 🚨 Critical | Wrong settlement | Verify baseAmount recalc when fee changes |
| Fee not converted at capture | ⚠️ High | Wrong currency display | Check fee conversion in capture flow |
| Missing denomination factor | 🚨 Critical | Wrong amounts (100x off) | Verify denominationFactor in all conversions |
| Floor/round instead of ceiling | ⚠️ High | Merchant loses money | Check for math.Ceil() usage |
| Three-decimal not rounded | 🚨 Critical | Payment fails | Check KWD/OMR/BHD rounding to nearest 10 |
| Division by zero | 🚨 Critical | Service crash | Check for zero protection before division |
| Base currency from request | 🚨 Critical | Regulatory violation | Derive from merchant country, not request |
| INR-INR not handled | ⚠️ High | Unnecessary conversion | Check for same-currency special case |
| Silent zero on error | ⚠️ High | Masked bugs | Return errors, not zero |

---

## Related Documentation

- [CFB Fee Handling](#cfb-fee-handling-patterns) - Correct fee patterns
- [Currency Validation](#currency-validation-patterns) - Currency consistency
- [Exchange Rate Patterns](#exchange-rate-patterns) - Rate application
- [Lifecycle Transitions](#lifecycle-transitions) - Stage transitions

---

## Testing Checklist

When reviewing cross-border code, verify:

- [ ] Fee subtracted before markdown (if CFB)
- [ ] Fee converted at original rate (not markdown)
- [ ] Fee recalculated when changed
- [ ] Currencies matched before arithmetic
- [ ] Fee converted at capture (MCC CFB)
- [ ] Denomination factor included
- [ ] Ceiling rounding used
- [ ] Three-decimal currencies rounded to 10
- [ ] Division by zero protected
- [ ] Base currency derived from merchant country
- [ ] INR-INR special case handled
- [ ] Errors returned (not silent zeros)
