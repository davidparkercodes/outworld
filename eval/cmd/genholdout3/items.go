package main

func itemGroups() [][]def {
	return [][]def{
		itemsCodeA, itemsCodeB, itemsCodeC,
		itemsWorkA, itemsWorkB,
		itemsPersonalA, itemsPersonalB,
		itemsDataA, itemsDataB,
		itemsIntlA, itemsIntlB, itemsIntlC,
		itemsSecretsA, itemsSecretsB,
		itemsNegatives,
		longGenerated(), longWritten(),
	}
}
