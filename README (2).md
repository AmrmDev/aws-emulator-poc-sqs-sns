# AWS Emulator POC --- SNS → SQS → Go Consumer

> Demonstração local de mensageria assíncrona usando **Amazon SNS**,
> **Amazon SQS**, **Go** e o **AWS Emulator** executado em Docker.

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)
![AWS](https://img.shields.io/badge/AWS-SNS%20%7C%20SQS-FF9900?logo=amazonaws&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Desktop-2496ED?logo=docker&logoColor=white)

## Visão geral

Esta POC demonstra um fluxo simples de publicação e consumo de eventos:

1.  Uma aplicação Go publica um evento JSON em um tópico do **Amazon
    SNS**.
2.  O SNS entrega automaticamente a notificação para uma fila do
    **Amazon SQS**, por meio de uma subscription.
3.  Uma segunda aplicação Go recebe a mensagem usando **Long Polling**.
4.  O consumer interpreta o envelope de notificação do SNS, extrai o
    JSON original e simula o processamento do evento.
5.  Após o processamento bem-sucedido, o consumer exclui a mensagem da
    fila com `DeleteMessage`.

Os serviços AWS são executados localmente pelo AWS Emulator em um
container Docker. A aplicação Go e a AWS CLI acessam o endpoint local
por HTTPS.

## Arquitetura

``` mermaid
flowchart LR
    A[Go Publisher] -->|Publish JSON| B[Amazon SNS]
    B -->|Subscription: SQS| C[Amazon SQS]
    C -->|ReceiveMessage / Long Polling| D[Go Consumer]
    D -->|Processa evento| E[DeleteMessage]
```

### Recursos utilizados

  ------------------------------------------------------------------------------------------------------------------------------
  Recurso                 Nome                       Identificador
  ----------------------- -------------------------- ---------------------------------------------------------------------------
  Região                  `us-east-1`                ---

  Endpoint local          `https://127.0.0.1:9297`   ---

  SNS Topic               `poc-payment-events`       `arn:aws:sns:us-east-1:000000000000:poc-payment-events`

  SQS Queue               `poc-payment-processing`   `https://sqs.us-east-1.amazonaws.com/000000000000/poc-payment-processing`

  Subscription            SNS → SQS                  Protocolo `sqs`
  ------------------------------------------------------------------------------------------------------------------------------

O account ID `000000000000` é o identificador local apresentado pelo
Emulator, não uma conta AWS real.

## Tecnologias

-   Go 1.24
-   AWS SDK for Go v2
-   Amazon SNS
-   Amazon SQS
-   AWS Emulator
-   Docker Desktop
-   WSL 2
-   AWS CLI
-   Windows PowerShell

## 1. Preparação do ambiente Docker

O ambiente foi configurado no Windows usando Docker Desktop com backend
Linux baseado em WSL 2.

### 1.1. Atualizar o WSL

No PowerShell:

``` powershell
wsl --update
wsl --shutdown
```

O `wsl --shutdown` encerra as distribuições e reinicia o ambiente WSL na
próxima inicialização.

### 1.2. Verificar o Docker

``` powershell
docker info
```

O comando deve apresentar informações do **Client** e do **Server**. Se
aparecer um erro de conexão com `dockerDesktopLinuxEngine`, o Docker
Engine ainda não está disponível. Abra o Docker Desktop e aguarde a
inicialização antes de continuar.

Também é possível verificar o estado das distribuições WSL:

``` powershell
wsl --list --verbose
```

Na configuração utilizada nesta POC, a distribuição `docker-desktop`
aparece com versão 2 quando o Docker Desktop está inicializado.

## 2. Construção da imagem do AWS Emulator

Foi criada uma imagem Docker própria a partir da imagem mínima do Amazon
Linux 2023. Durante o build, o pacote RPM do AWS Emulator é baixado e
instalado.

Crie uma pasta para o ambiente:

``` powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\aws-emulator-poc"
Set-Location "$env:USERPROFILE\aws-emulator-poc"
```

Crie um arquivo chamado `Dockerfile` (sem extensão `.txt`).

Exemplo de Dockerfile correspondente à imagem construída nesta POC:

``` dockerfile
FROM public.ecr.aws/amazonlinux/amazonlinux:2023-minimal

ADD https://downloads.aws.com/aws-emulator/packages/aws-emulator/LTS/latest/aws-emulator-x86_64.rpm /tmp/aws-emulator.rpm

RUN rpm -i /tmp/aws-emulator.rpm \
    && rm /tmp/aws-emulator.rpm \
    && mkdir -p /home/emulator \
    && chown 1000 /home/emulator
```

> **Observação:** o conteúdo acima reproduz a base, o pacote e as etapas
> de instalação observadas nos logs do build. Caso o Dockerfile original
> tenha instruções adicionais, mantenha-as. O arquivo precisa se chamar
> exatamente `Dockerfile`, e não `Dockerfile.txt`.

Se o arquivo tiver sido criado como `Dockerfile.txt`, renomeie:

``` powershell
Rename-Item Dockerfile.txt Dockerfile
```

Construa a imagem:

``` powershell
docker build -t aws-emulator .
```

Confira se ela foi criada:

``` powershell
docker images
```

Nos logs desta execução, a imagem `aws-emulator:latest` foi criada com
sucesso. O tamanho exibido foi aproximadamente 2,03 GB de uso em disco e
577 MB de conteúdo.

## 3. Inicialização do AWS Emulator

O Emulator utiliza uma pasta persistente no perfil do usuário para
armazenar arquivos de configuração, credenciais e certificados.

Crie o diretório:

``` powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\.aws\emulator"
```

Inicie o container:

``` powershell
$port = 9297

docker run --rm --memory=2g --memory-swap=2g `
  -p "127.0.0.1:${port}:${port}" `
  -v "$env:USERPROFILE\.aws\emulator:/home/emulator/.aws/emulator" `
  aws-emulator -p $port --telemetry off
```

### O que cada opção faz

  -----------------------------------------------------------------------
  Opção                               Finalidade
  ----------------------------------- -----------------------------------
  `--rm`                              Remove o container quando ele é
                                      encerrado

  `--memory=2g`                       Limita a memória do container a 2
                                      GB

  `--memory-swap=2g`                  Define o limite combinado de
                                      memória e swap em 2 GB

  `-p 127.0.0.1:9297:9297`            Publica a porta local somente na
                                      interface de loopback

  `-v ...`                            Persiste os arquivos do Emulator no
                                      Windows

  `-p 9297`                           Inicia o Emulator na porta 9297

  `--telemetry off`                   Desabilita a telemetria
  -----------------------------------------------------------------------

Mantenha esse terminal aberto enquanto estiver executando a POC.

### 3.1. Verificar a inicialização

O log de inicialização deve indicar o endpoint:

``` text
endpoint_url: https://127.0.0.1:9297
health_check_url: https://127.0.0.1:9297/__admin/health
```

O health check pode ser validado no PowerShell:

``` powershell
curl.exe --cacert "$env:AWS_CA_BUNDLE" `
  https://127.0.0.1:9297/__admin/health
```

Resultado esperado:

``` text
ok
```

O Emulator gera e utiliza sua própria CA. O caminho do certificado deve
ser configurado conforme os arquivos gerados pelo Emulator. Nos testes
desta máquina, `AWS_CA_BUNDLE` foi utilizado para apontar para o
certificado CA correspondente.

## 4. Configuração da AWS CLI

Configure as variáveis de ambiente no PowerShell:

``` powershell
$env:AWS_REGION = "us-east-1"
$env:AWS_ENDPOINT_URL = "https://127.0.0.1:9297"

$env:SNS_TOPIC_ARN = "arn:aws:sns:us-east-1:000000000000:poc-payment-events"

$env:SQS_QUEUE_URL = "https://sqs.us-east-1.amazonaws.com/000000000000/poc-payment-processing"
```

Para facilitar os comandos manuais, também defina:

``` powershell
$endpoint = "https://127.0.0.1:9297"

$topicArn = "arn:aws:sns:us-east-1:000000000000:poc-payment-events"

$queueUrl = "https://sqs.us-east-1.amazonaws.com/000000000000/poc-payment-processing"
```

As variáveis definidas com `$env:` ficam disponíveis nos processos
iniciados a partir daquela sessão do PowerShell. Se abrir outro
terminal, configure-as novamente.

### 4.1. Validar a identidade local

``` powershell
aws sts get-caller-identity `
  --endpoint-url $endpoint `
  --region us-east-1
```

O Emulator utiliza credenciais locais. A identidade retornada não
representa uma identidade IAM de uma conta AWS real.

## 5. Criação e validação dos recursos SNS e SQS

Os recursos desta POC são:

-   Um tópico SNS: `poc-payment-events`.
-   Uma fila SQS: `poc-payment-processing`.
-   Uma subscription ligando o tópico SNS à fila SQS.

Os ARNs e a URL apresentados abaixo são os identificadores utilizados
nesta execução.

### 5.1. Conferir o tópico SNS

``` powershell
aws sns list-topics `
  --region us-east-1 `
  --endpoint-url $endpoint
```

Resultado observado:

``` json
{
  "Topics": [
    {
      "TopicArn": "arn:aws:sns:us-east-1:000000000000:poc-payment-events"
    }
  ]
}
```

### 5.2. Conferir a fila SQS

``` powershell
aws sqs list-queues `
  --region us-east-1 `
  --endpoint-url $endpoint
```

Resultado observado:

``` json
{
  "QueueUrls": [
    "https://sqs.us-east-1.amazonaws.com/000000000000/poc-payment-processing"
  ]
}
```

### 5.3. Conferir a subscription

``` powershell
aws sns list-subscriptions-by-topic `
  --topic-arn $topicArn `
  --region us-east-1 `
  --endpoint-url $endpoint
```

Resultado observado:

``` json
{
  "Subscriptions": [
    {
      "SubscriptionArn": "arn:aws:sns:us-east-1:000000000000:poc-payment-events:63e7451b-b63e-4936-b099-94c6d82d172e",
      "Owner": "000000000000",
      "Protocol": "sqs",
      "Endpoint": "arn:aws:sqs:us-east-1:000000000000:poc-payment-processing",
      "TopicArn": "arn:aws:sns:us-east-1:000000000000:poc-payment-events"
    }
  ]
}
```

Isso confirma a existência da subscription com protocolo `sqs` e o
tópico e fila esperados.

### 5.4. Permissão da fila para receber notificações do SNS

A fila precisa permitir que o serviço SNS execute `sqs:SendMessage`.
Durante a configuração, o Emulator registrou este aviso:

``` text
the subscribed queue's policy does not allow this service to send it a message,
so the notification is withheld
```

Esse aviso indica que a subscription existia, mas a política da fila não
permitia a entrega.

Uma política de fila para esse cenário pode permitir `sqs:SendMessage`
somente quando a origem for o tópico esperado:

Crie `sqs-queue-policy.json`:

``` json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowSNSPublishToQueue",
      "Effect": "Allow",
      "Principal": {
        "Service": "sns.amazonaws.com"
      },
      "Action": "sqs:SendMessage",
      "Resource": "arn:aws:sqs:us-east-1:000000000000:poc-payment-processing",
      "Condition": {
        "ArnEquals": {
          "aws:SourceArn": "arn:aws:sns:us-east-1:000000000000:poc-payment-events"
        }
      }
    }
  ]
}
```

Aplique a política à fila:

``` powershell
$attributes = @{
    Policy = (Get-Content .\sqs-queue-policy.json -Raw)
} | ConvertTo-Json -Compress

[System.IO.File]::WriteAllText(
    "$PWD\queue-attributes.json",
    $attributes,
    [System.Text.UTF8Encoding]::new($false)
)

aws sqs set-queue-attributes `
  --queue-url $queueUrl `
  --attributes file://queue-attributes.json `
  --region us-east-1 `
  --endpoint-url $endpoint
```

> A política acima documenta a permissão necessária para a entrega SNS →
> SQS. Valide o comportamento no Emulator utilizado; a sintaxe e o
> suporte a políticas podem variar entre implementações locais.

## 6. Implementação Go

A aplicação utiliza o AWS SDK for Go v2 para se comunicar com SNS e SQS.

### Estrutura do projeto

``` text
aws-emulator-poc-sqs-sns/
├── cmd/
│   ├── publisher/
│   │   └── main.go
│   └── consumer/
│       └── main.go
├── config/
│   └── config.go
├── internal/
│   └── messaging/
│       ├── sns.go
│       └── sqs.go
├── go.mod
└── README.md
```

### Publisher

O Publisher:

1.  Cria um evento de pagamento.
2.  Serializa o evento em JSON.
3.  Publica o payload no tópico SNS.
4.  Exibe o `MessageId` retornado pelo serviço.

Exemplo do evento publicado:

``` json
{
  "paymentId": "pay-go-001",
  "amount": 150.5,
  "currency": "BRL",
  "method": "PIX",
  "status": "PENDING",
  "createdAt": "2026-11-20T13:17:57Z"
}
```

O campo `createdAt` é gerado em UTC durante a execução; o horário e os
milissegundos variam.

### Consumer

O Consumer executa continuamente o seguinte fluxo:

1.  Chama `ReceiveMessage` com `WaitTimeSeconds: 20`.
2.  Recebe mensagens da fila.
3.  Desserializa o corpo recebido como um envelope SNS.
4.  Extrai o conteúdo original do campo `Message`.
5.  Desserializa o evento de pagamento.
6.  Simula o processamento imprimindo os dados.
7.  Executa `DeleteMessage` após o processamento bem-sucedido.

O Long Polling reduz chamadas vazias ao aguardar mensagens por até 20
segundos em cada requisição de recebimento.

### SNS Notification Envelope

Quando o SNS entrega uma notificação à fila, o corpo recebido pelo SQS
não é apenas o JSON do pagamento. Por padrão, ele é encapsulado em uma
estrutura de notificação semelhante a esta:

``` json
{
  "Type": "Notification",
  "MessageId": "84f8ece1-745a-472c-8f9d-4431e9e6cf0a",
  "TopicArn": "arn:aws:sns:us-east-1:000000000000:poc-payment-events",
  "Message": "{\"paymentId\":\"pay-go-001\",\"amount\":150.5,\"currency\":\"BRL\",\"method\":\"PIX\",\"status\":\"PENDING\"}",
  "Timestamp": "2026-11-20T13:17:57.067Z"
}
```

O campo `Message` contém o JSON original publicado pelo Publisher. Por
isso, o Consumer primeiro interpreta o envelope SNS e depois
desserializa o conteúdo de `Message`.

### Exclusão da mensagem

O SQS não remove uma mensagem automaticamente quando ela é recebida. O
recebimento a torna temporariamente invisível durante o Visibility
Timeout.

Depois de processar a mensagem com sucesso, o Consumer chama:

``` go
c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
    QueueUrl:      aws.String(c.queueURL),
    ReceiptHandle: message.ReceiptHandle,
})
```

O `ReceiptHandle` é obtido no recebimento daquela mensagem e é utilizado
para solicitar sua exclusão.

Se o processamento falhar, o Consumer não executa `DeleteMessage`. A
mensagem poderá voltar a ficar disponível após o Visibility Timeout.

## 7. Executando a POC

Antes de executar, confirme:

-   Docker Desktop está ativo.
-   O container do AWS Emulator está em execução.
-   O health check retorna `ok`.
-   O tópico SNS existe.
-   A fila SQS existe.
-   A subscription SNS → SQS existe.
-   A fila permite que o SNS envie mensagens.
-   As variáveis de ambiente estão configuradas no terminal.

### Terminal 1 --- iniciar o Consumer

Na raiz do projeto Go:

``` bash
go run ./cmd/consumer
```

Saída inicial esperada:

``` text
2026/11/20 10:17:44 SQS CONSUMER STARTED
```

O processo permanece aguardando mensagens.

### Terminal 2 --- executar o Publisher

Na raiz do projeto Go:

``` bash
go run ./cmd/publisher
```

O Publisher publica o evento no SNS e exibe o identificador retornado.

O Consumer deverá receber a notificação, extrair o evento e processá-lo.

### Exemplo de log real da aplicação

A execução validada nesta POC apresentou:

``` text
2026/11/20 10:17:44 SQS CONSUMER STARTED
2026/11/20 10:17:44

========== SQS CONSUMER ==========
SNS Notification received!
SNS Message ID: 84f8ece1-745a-472c-8f9d-4431e9e6cf0a
SNS Topic ARN: arn:aws:sns:us-east-1:000000000000:poc-payment-events
SNS Timestamp: 2026-11-20T13:17:57.067Z
-----------------------------------
Payment event processed:
map[amount:150.5 createdAt:2026-11-20T13:17:57.0251837Z currency:BRL method:PIX paymentId:pay-go-001 status:PENDING]
-----------------------------------
Message deleted successfully!
===================================
```

Esse log demonstra a entrega da notificação SNS, a extração e o
processamento do evento e a exclusão da mensagem no SQS.

## 8. Testes manuais usando AWS CLI

Além da aplicação Go, o fluxo também foi explorado diretamente pela AWS
CLI.

Todos os comandos abaixo utilizam o endpoint local.

### 8.1. Publicar uma mensagem no SNS

``` powershell
aws sns publish `
  --topic-arn $topicArn `
  --message "Payment event from SNS!" `
  --region us-east-1 `
  --endpoint-url $endpoint
```

Resposta observada:

``` json
{
  "MessageId": "4ff477ee-f9b9-415e-8fe6-53ab1a754c97"
}
```

### 8.2. Receber mensagens no SQS

``` powershell
aws sqs receive-message `
  --queue-url $queueUrl `
  --max-number-of-messages 5 `
  --region us-east-1 `
  --endpoint-url $endpoint
```

A inspeção retornou dois tipos de mensagem durante os testes:

-   Uma mensagem de teste simples: `Hello from AWS Emulator!`.
-   Uma notificação SNS contendo o envelope `Type`, `MessageId`,
    `TopicArn`, `Message` e `Timestamp`.

Isso permitiu observar diretamente como o SNS encapsula a mensagem antes
de entregá-la ao SQS.

### 8.3. Excluir uma mensagem

Para excluir uma mensagem manualmente, utilize o `ReceiptHandle`
retornado pela chamada de `receive-message`:

``` powershell
aws sqs delete-message `
  --queue-url $queueUrl `
  --receipt-handle "<RECEIPT_HANDLE_RETORNADO_PELO_RECEIVE>" `
  --region us-east-1 `
  --endpoint-url $endpoint
```

O `ReceiptHandle` não é um valor fixo: use o valor exato retornado para
aquela entrega. Um valor fictício, um handle antigo ou um handle copiado
incorretamente pode resultar em `ReceiptHandleIsInvalid`.

### 8.4. Consultar a quantidade de mensagens

``` powershell
aws sqs get-queue-attributes `
  --queue-url $queueUrl `
  --attribute-names ApproximateNumberOfMessages ApproximateNumberOfMessagesNotVisible `
  --region us-east-1 `
  --endpoint-url $endpoint
```

Os atributos permitem observar:

-   `ApproximateNumberOfMessages`: mensagens aproximadamente disponíveis
    para recebimento.
-   `ApproximateNumberOfMessagesNotVisible`: mensagens temporariamente
    invisíveis, por exemplo, após serem recebidas por um consumer.

Após o consumo e a exclusão, os valores observados foram:

``` json
{
  "Attributes": {
    "ApproximateNumberOfMessagesNotVisible": "0",
    "ApproximateNumberOfMessages": "0"
  }
}
```

### 8.5. Limpar a fila durante os testes

Para descartar mensagens de testes anteriores:

``` powershell
aws sqs purge-queue `
  --queue-url $queueUrl `
  --region us-east-1 `
  --endpoint-url $endpoint
```

A operação de purge remove as mensagens da fila e não deve ser usada
enquanto mensagens importantes precisarem ser preservadas.

## 9. Problemas encontrados e aprendizados

### Docker Engine indisponível

**Sintoma:** `docker info` apresentava erro de conexão com
`dockerDesktopLinuxEngine`.

**Investigação:** foram verificados os serviços do Windows, o estado do
WSL e a inicialização do Docker Desktop.

**Resolução:** atualizar o WSL, executar `wsl --shutdown` e aguardar o
Docker Desktop inicializar corretamente.

### Dockerfile não encontrado

**Sintoma:**

``` text
failed to read dockerfile: open Dockerfile: no such file or directory
```

**Causa:** o arquivo havia sido criado como `Dockerfile.txt`.

**Resolução:**

``` powershell
Rename-Item Dockerfile.txt Dockerfile
```

Depois disso, o build foi concluído.

### Política da fila bloqueando a entrega SNS → SQS

**Sintoma:** o Emulator registrou que a política da fila não permitia
que o SNS enviasse mensagens.

**Aprendizado:** criar uma subscription não garante, por si só, que a
fila autorize a publicação do SNS. A permissão `sqs:SendMessage` precisa
estar disponível para o tópico de origem.

### `ReceiptHandleIsInvalid`

**Sintoma:** o comando `delete-message` retornou:

``` text
ReceiptHandleIsInvalid
```

**Causa no teste:** primeiro foi utilizado um texto de exemplo no lugar
do ReceiptHandle real. Depois, foi utilizado o handle retornado pelo
recebimento.

**Aprendizado:** o `ReceiptHandle` deve ser copiado da resposta de
recebimento correspondente. Não se deve reutilizar um placeholder ou
presumir que um handle antigo continua válido.

### Mensagem de teste adicional na fila

Durante a inspeção manual, havia uma mensagem simples
(`Hello from AWS Emulator!`) além da notificação SNS. Isso explica por
que a fila ainda apresentava uma mensagem após a exclusão da
notificação. A fila foi posteriormente limpa com `purge-queue` para
deixar os testes seguintes em estado conhecido.

## 10. Escopo da POC

### O que foi implementado

-   [x] Inicialização local do AWS Emulator em Docker.
-   [x] Configuração do endpoint HTTPS local.
-   [x] Tópico SNS.
-   [x] Fila SQS.
-   [x] Subscription SNS → SQS.
-   [x] Publicação de evento JSON com Go.
-   [x] Entrega automática da notificação SNS ao SQS.
-   [x] Recebimento com Long Polling.
-   [x] Desserialização do SNS Notification Envelope.
-   [x] Extração do evento original.
-   [x] Exclusão com `DeleteMessage` após processamento bem-sucedido.
-   [x] Testes manuais com AWS CLI.

### O que não faz parte desta implementação

-   Banco de dados ou persistência de pagamentos.
-   Idempotência.
-   Dead Letter Queue (DLQ).
-   Estratégia de retry customizada.
-   Regras de negócio reais de pagamentos.
-   Observabilidade distribuída ou monitoramento avançado.
-   Infraestrutura de produção.

## 11. Conclusão

A POC demonstra, localmente, o padrão de mensageria
**publish/subscribe** com SNS e processamento assíncrono com SQS.

O Publisher não precisa conhecer o Consumer nem publicar diretamente na
fila. Ele publica no tópico SNS, e a subscription faz a integração com o
SQS. O Consumer fica responsável por buscar as mensagens, interpretar o
envelope, processar o evento e confirmar o consumo por meio de
`DeleteMessage`.

A execução local com AWS Emulator permite experimentar esse fluxo usando
Docker, AWS CLI e AWS SDK for Go v2, sem precisar apontar a aplicação
para recursos de uma conta AWS real.

------------------------------------------------------------------------

## Referências úteis

-   [AWS Emulator --- documentação
    oficial](https://docs.aws.amazon.com/)

-   [Amazon SNS --- documentação](https://docs.aws.amazon.com/sns/)

-   [Amazon SQS --- documentação](https://docs.aws.amazon.com/sqs/)

-   [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)

------------------------------------------------------------------------

**Projeto:** `aws-emulator-poc-sqs-sns`\
**Região configurada:** `us-east-1`\
**Endpoint local:** `https://127.0.0.1:9297`
