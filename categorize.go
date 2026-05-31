package main

import "strings"

type categoryRule struct {
	name     string
	keywords []string
}

var categoryRules = []categoryRule{
	{"Supermercados", []string{
		"continente", "pingo doce", "lidl", "aldi", "minipreco", "mini preco",
		"intermarche", "intermarch", "jumbo", "mercadona", "modelo", "supermercado",
		"el corte ingles", "pingo",
	}},
	{"Combustível", []string{
		"galp", " bp ", "repsol", "cepsa", "shell", "prio", "combustivel",
		"bomba gasolina", "posto combustivel",
	}},
	{"Transportes", []string{
		"cp comboios", "comboios de portugal", "metro de lisboa", "metro do porto",
		"carris", "stcp", "fertagus", "transtejo", "soflusa",
		"uber", "bolt ", "cabify", "autoestrada", "via verde", "portagem",
		"rede expressos", "flixbus", "transporte",
	}},
	{"Telecomunicações", []string{
		"nos ", " nos", "meo ", " meo", "vodafone", "nowo", "nos comunicacoes",
		"meo fibra", "telecomunicacoes",
	}},
	{"Energia & Utilities", []string{
		"edp ", " edp", "endesa", "galp energia", "iberdrola", "epal", " agua",
		"gas natural", "naturgy", "egf", "aguas de", "smas",
	}},
	{"Farmácias & Saúde", []string{
		"farmacia", "wells ", "dr. well", "saude viva", "clinica", "medico",
		"hospital", "laboratorio", "analises", "dentista", "otica",
		"optometrista", "veterinario", " vet ", "apotheka", "holon",
	}},
	{"Restauração", []string{
		"restaurante", "cafe ", "pastelaria", "padaria", "mcdonalds", "mcdonald",
		"burger king", "kfc", "nandos", "pizza hut", "dominos", "subway",
		"pizza", "sushi", "tasca", "taberna", "snack bar", "gelataria",
		"cervejaria", "marisqueira", "churrasqueira",
	}},
	{"Vestuário & Calçado", []string{
		"zara", "h&m", "primark", "bershka", "pull&bear", "mango",
		"stradivarius", "sport zone", "decathlon", "roupa", "calcado",
		"sapatos", "massimo dutti", "cortefiel",
	}},
	{"Lazer & Cultura", []string{
		"fnac", "worten", "cinema", "museu", "teatro", "bilhete", "ticketline",
		"spotify", "netflix", "hbo", "disney", "amazon prime", "google play",
		"steam", "playstation", "xbox", "apple ", "bol.pt",
	}},
	{"Habitação", []string{
		"renda", "condominio", "seguro casa", "camara municipal", "imt ",
		"imobiliaria", "arrendamento",
	}},
	{"Seguros", []string{
		"seguro", "fidelidade", "ageas", "allianz", "generali", "zurich",
		"axa ", "tranquilidade", "lusitania",
	}},
	{"Educação", []string{
		"escola", "colegio", "universidade", "iscte", "ispa", "propina",
		"livros", "papelaria", "fnac livros",
	}},
	{"Viagens", []string{
		"tap ", "ryanair", "easyjet", "wizz", "booking", "airbnb",
		"hotel", "hostel", "turismo", "aeroporto", "airport", "flightgift",
	}},
	{"Fitness & Bem-estar", []string{
		"ginasio", "holmes place", "fitness", "academia", "yoga", "pilates",
		"spa", "estetica", "cabeleireiro", "barbeiro",
	}},
	{"Bancos & Financeiro", []string{
		"santander", "caixa geral", "cgd", "millennium", "novobanco",
		"montepio", " bpi ", " bcp ", "mbway", "multibanco",
		"transferencia", "comissao", "juro",
	}},
	{"Serviços Públicos", []string{
		"at.gov", "irs ", "seguranca social", "imtt", "governo",
		"ministerio", "ss direct", "autoridade tributaria",
	}},
}

var normReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a",
	"é", "e", "ê", "e", "è", "e",
	"í", "i", "ï", "i",
	"ó", "o", "ô", "o", "õ", "o",
	"ú", "u", "ü", "u",
	"ç", "c",
)

func categorize(description string) string {
	lower := normReplacer.Replace(strings.ToLower(description))
	for _, rule := range categoryRules {
		for _, kw := range rule.keywords {
			if strings.Contains(lower, kw) {
				return rule.name
			}
		}
	}
	return "Outros"
}
