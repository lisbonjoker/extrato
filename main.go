package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

var buildVersion = ""

func getVersion() string {
	if buildVersion != "" {
		return buildVersion
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

var outputFlag string

var rootCmd = &cobra.Command{
	Use:   "extrato <ficheiro>",
	Short: "Analisador de extratos bancários Santander Portugal",
	Long: `extrato — Analisador de extratos bancários Santander Portugal.

Analisa extratos em formato CSV e OFX e gera um relatório HTML
interativo com gráficos, estatísticas e categorização automática
das transações por comerciante e tipo de despesa.

Formatos suportados:
  .csv   Exportação CSV do Santander NetBanking
  .ofx   Exportação OFX/QFX do Santander

Exemplos:
  extrato statement.csv
  extrato statement.ofx
  extrato statement.csv --output relatorio.html
  extrato statement.ofx -o ~/relatorios/junho.html`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile := args[0]

		data, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("erro ao ler ficheiro: %w", err)
		}

		ext := strings.ToLower(filepath.Ext(inputFile))

		var txns []Transaction
		switch ext {
		case ".csv":
			txns, err = parseCSV(data)
		case ".ofx", ".qfx":
			txns, err = parseOFX(data)
		default:
			return fmt.Errorf("formato não suportado: %s (use .csv ou .ofx)", ext)
		}
		if err != nil {
			return fmt.Errorf("erro ao analisar ficheiro: %w", err)
		}

		if len(txns) == 0 {
			return fmt.Errorf("nenhuma transação encontrada no ficheiro")
		}

		for i := range txns {
			txns[i].Category = categorize(txns[i].Description)
		}

		out := outputFlag
		if out == "" {
			base := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
			out = base + ".html"
		}

		if err := generateReport(txns, out); err != nil {
			return fmt.Errorf("erro ao gerar relatório: %w", err)
		}

		fmt.Printf("Relatório gerado: %s (%d transações)\n", out, len(txns))
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Mostrar a versão instalada",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("extrato " + getVersion())
	},
}

func init() {
	rootCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Ficheiro de saída HTML (por omissão: <ficheiro>.html)")
	rootCmd.AddCommand(versionCmd)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "help [comando]",
		Short:  "Ajuda sobre qualquer comando",
		Hidden: true,
		Run: func(c *cobra.Command, args []string) {
			cmd, _, err := c.Root().Find(args)
			if cmd == nil || err != nil {
				c.Printf("Comando desconhecido: %q\n", args)
			} else {
				_ = cmd.Help()
			}
		},
	})
	rootCmd.SetUsageTemplate(`Utilização:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [comando]{{end}}{{if .HasAvailableSubCommands}}

Comandos disponíveis:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Opções:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}

Utilize "{{.CommandPath}} [comando] --help" para mais informação.
`)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
