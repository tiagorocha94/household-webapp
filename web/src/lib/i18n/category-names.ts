// How the fixed categories and budget groups are named on screen in Portuguese. The name itself
// (English, as in the CSV files) is the identifier and never changes; only what the person reads
// does. The category names are the same as the phone app's.
export const categoryNamesPt: Readonly<Record<string, string>> = {
  Restaurant: 'Restaurante',
  Mortgage: 'Crédito habitação',
  Insurance: 'Seguros',
  Security: 'Segurança',
  Water: 'Água',
  Electricity: 'Eletricidade',
  Groceries: 'Supermercado',
  Health: 'Saúde',
  Telecom: 'Telecomunicações',
  Investments: 'Investimentos',
  Objectives: 'Objetivos',
  Savings: 'Poupança',
  Leisure: 'Lazer',
  'Bank Fees': 'Comissões bancárias',
  Transportation: 'Transportes',
  Education: 'Educação',
  Gifts: 'Presentes',
  Miscellaneous: 'Diversos',
  Salary: 'Salário',
  Interest: 'Juros',
  Bonus: 'Bónus',
  'Meal Allowance': 'Subsídio de refeição',
  Gift: 'Presente',
  Others: 'Outros',
};

export const groupNamesPt: Readonly<Record<string, string>> = {
  Essential: 'Essencial',
  Investment: 'Investimento',
  'Goals & Savings': 'Objetivos e poupança',
  Fun: 'Lazer',
};
