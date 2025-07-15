package payment

type PaymentWorker struct {
    r Repository
}

func NewPaymentWorker(repository Repository) *PaymentWorker {
	return &PaymentWorker{
        r: repository,
    }
}

func (w *PaymentWorker) Run() error {
	// implementation
	return nil
}

func (w *PaymentWorker) GetHealthStatus() error {
	// implementation
	return nil
}
