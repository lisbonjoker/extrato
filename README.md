# extrato

Analisador de extratos bancários Santander Portugal em linha de comandos.

Gera um relatório HTML interativo e autocontido a partir de exportações CSV e OFX do Santander NetBanking, com gráficos, estatísticas e categorização automática de transações.

## Demonstração

```
$ extrato statement.csv
Relatório gerado: statement.html (247 transações)
```

O relatório inclui:

| Secção | Descrição |
|---|---|
| **KPI Cards** | Entradas, Saídas, Saldo Líquido, Nº Transações |
| **Gráfico de Barras** | Receitas vs Despesas mensais |
| **Gráfico Donut** | Despesas por categoria |
| **Gráfico de Área** | Evolução do saldo ao longo do tempo |
| **Top 10 Comerciantes** | Com barra de proporção e contagem |
| **Transações Recentes** | Últimas 25 transações com categorias |

## Instalação

### Via `go install`

```bash
go install github.com/lisbonjoker/extrato@latest
```

### A partir do código fonte

```bash
git clone https://github.com/lisbonjoker/extrato.git
cd extrato
make install
```

## Utilização

```bash
# CSV exportado do Santander NetBanking
extrato statement.csv

# OFX/QFX
extrato statement.ofx

# Escolher o nome do ficheiro de saída
extrato statement.csv --output relatorio-junho-2024.html
extrato statement.csv -o ~/relatorios/junho.html

# Ver versão
extrato version
```

## Formatos suportados

### CSV

Exportação do Santander NetBanking em CSV (separador `;` ou `,`). O cabeçalho é detectado automaticamente — são suportadas variantes de nomes de colunas como `Data`, `Descrição`, `Débito`, `Crédito`, `Saldo`, etc.

Exemplo de ficheiro compatível:

```
Data;Descrição;Débito;Crédito;Saldo
01-06-2024;CONTINENTE MODELO CASCAIS;45,80;;2.543,20
03-06-2024;GALP COMBUSTÍVEIS;60,00;;2.483,20
05-06-2024;TRANSFERÊNCIA RECEBIDA;;1.500,00;3.983,20
```

### OFX / QFX

Formato Open Financial Exchange, exportável do Santander NetBanking e compatível com a maioria dos bancos portugueses.

## Categorias

As transações são categorizadas automaticamente por palavras-chave no campo descrição:

| Categoria | Exemplos |
|---|---|
| Supermercados | Continente, Pingo Doce, Lidl, Aldi, Intermarché |
| Combustível | Galp, BP, Repsol, Cepsa, Shell |
| Transportes | CP Comboios, Metro, Carris, Uber, Via Verde |
| Telecomunicações | NOS, MEO, Vodafone, Nowo |
| Energia & Utilities | EDP, Endesa, EPAL, Águas de... |
| Farmácias & Saúde | Farmácias, Wells, Clínicas, Dentistas |
| Restauração | Restaurantes, Cafés, McDonald's, Pizza |
| Vestuário & Calçado | Zara, H&M, Primark, Decathlon |
| Lazer & Cultura | FNAC, Cinema, Netflix, Spotify |
| Habitação | Rendas, Condomínio |
| Seguros | Fidelidade, Ageas, Allianz |
| Educação | Propinas, Escolas, Universidades |
| Viagens | TAP, Ryanair, Booking, Airbnb |
| Fitness & Bem-estar | Ginásios, Holmes Place |
| Bancos & Financeiro | Comissões, Transferências, MBWay |
| Serviços Públicos | AT, Segurança Social |
| Outros | Tudo o resto |

## Makefile

```bash
make build    # Compilar o binário (./extrato)
make install  # Instalar com go install
make run ARGS="statement.csv"  # Compilar e executar
make clean    # Remover binário
```

## Características

- **Autocontido** — o HTML gerado não precisa de ligação à internet para abrir (Chart.js carregado via CDN na geração, ou pode guardar offline)
- **Privacidade** — todo o processamento é feito localmente, nenhum dado é enviado para qualquer servidor
- **Tema escuro** — interface dark com vermelho Santander (`#EC0000`) como cor de destaque
- **Formatação pt-PT** — valores em `1.234,56 €`, datas em `DD/MM/AAAA`
- **Sem dependências externas** — apenas a biblioteca padrão do Go + Cobra para CLI

## Requisitos

- Go 1.21 ou superior

## Licença

MIT
