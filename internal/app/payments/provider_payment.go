package payments

func validateProviderPayment(
	providerPayment ProviderPayment,
	localPayment *Payment,
	expectedStatus Status,
) error {
	if localPayment.ProviderPaymentID == nil {
		return ErrProviderPaymentIDRequired
	}

	if providerPayment.ID != *localPayment.ProviderPaymentID {
		return ErrProviderPaymentIDMismatch
	}

	if providerPayment.Status != expectedStatus {
		return ErrPaymentStatusTransitionInvalid
	}

	if providerPayment.AmountCents != localPayment.AmountCents {
		return ErrPaymentAmountMismatch
	}

	if providerPayment.Currency != localPayment.Currency {
		return ErrPaymentCurrencyMismatch
	}

	return nil
}
