#!/usr/env powershell

function Write-Log() {
    [CmdletBinding()]
    param([Parameter(ValueFromPipeline=$true,Mandatory=$True)]$Text,[Switch]$NoSign,[Switch]$PlainText,[Switch]$PassThru)
    begin {$TT = @()}
    Process {$TT += ,$Text}
    END {
        $Blue = $(if ($WRITE_AP_LEGACY_COLORS){3}else{'Blue'})
        if ($TT.count -eq 1) {$TT = $TT[0]};$Text = $TT
        if ($text.count -gt 1 -or $text.GetType().Name -match "\[\]$") {return $Text |?{"$_"}| % {Write-AP $_ -NoSign:$NoSign -PlainText:$PlainText}}
        if (!$text -or $text -notmatch "^((?<NNL>x)|(?<NS>ns?)){0,2}(?<t>\>*)(?<s>[+\-!\*\#\@_])(?<w>.*)") {return $Text}
        $tb  = "    "*$Matches.t.length;
        $Col = @{'+'='2';'-'='12';'!'='14';'*'=$Blue;'#'='DarkGray';'@'='Gray';'_'='white'}[($Sign = $Matches.S)]
        if (!$Col) {Throw "Incorrect Sign [$Sign] Passed!"}
        $Sign = $(if ($NoSign -or $Matches.NS) {""} else {"[$Sign] "})
        $Data = "$tb$Sign$($Matches.W)";if (!$Data) {return}
        if ($PlainText) {return $Data}
        Write-Host -NoNewLine:$([bool]$Matches.NNL) -f $Col $Data
        if ($PassThru) {$Data}
    }
}

function Dep-Require([Parameter(Mandatory=$True)][Alias("Functionality","Library")][String]$Lib, [ScriptBlock]$OnFail={}, [Switch]$PassThru) {
    $Stat = $(switch -regex ($Lib.trim()) {
        "^dep:(.*)" {
            $pr = $matches[1]
            if ($IsLinux -or $IsMacOS) {
                try {which $pr 2>$null} catch {}
                break
            }
            try {where.exe $pr 2>$null} catch {}
        }
        default {Write-Log "!Invalid selector provided [$("$Lib".split(':')[0])]";throw 'BAD_SELECTOR'}
    })
    if (!$Stat) {$OnFail.Invoke()}
    if ($PassThru) {return $Stat}
}

function Clean-Up {
    Write-Log "!Stopping background jobs... $args"1
    # jobs -l
    $Script:PrA,$Script:PrB | ? {$_} | Stop-Process
    exit 1
}

Dep-Require "dep:kubectl" {Write-Log "!You're missing kubectl from your path, please install it / or add it to your PATH"; exit 1}
Dep-Require "dep:npm" {Write-Log "!You're missing npm from your path, please install it / or add it to your PATH"; exit 1}

$Namespace = if ($Env:NAMESPACE) {$Env:NAMESPACE} else {"kubeflow"}
Write-Log "*Preparing dev env for KFP frontend"

Write-Log "*Detecting api server pod names..."
$METADATA_ENVOY_POD = kubectl get pods -n $Namespace -l component=metadata-envoy -o=custom-columns=:.metadata.name --no-headers
if (!$METADATA_ENVOY_POD) {
    Write-Log ">!Couldn't get metadata envoy pod in namespace $Namespace, double check the cluster your kubectl talks to."
    exit 1
}
Write-Log ">+Metadata envoy pod is $METADATA_ENVOY_POD"

$PIPELINE_API_POD = kubectl get pods -n $Namespace -l app=ml-pipeline -o=custom-columns=:.metadata.name --no-headers
if (!$PIPELINE_API_POD) {
    Write-Log ">!Couldn't get pipeline api pod in namespace $Namespace, double check the cluster your kubectl talks to."
    exit 1
}
Write-Log ">+Ml pipeline PIPELINE_API api pod is $PIPELINE_API_POD"

Write-Log "*Compiling node server..."
pushd server;npm run build;popd

# Frontend dev server proxies api requests to node server listening to
# localhost:3001 (configured in frontend/package.json -> proxy field).
#
# Node server proxies requests further to localhost:3002 or localhost:9090
# based on what request it is.
#
# localhost:3002 port forwards to ml_pipeline api server pod.
# localhost:9090 port forwards to metadata_envoy pod.

Write-Log "*Starting to port forward backend apis..."
try {
    $Script:PrA = Start-Process -ws Hidden -PassThru kubectl "port-forward -n kubeflow $METADATA_ENVOY_POD 9090:9090"
    $Script:PrB = Start-Process -ws Hidden -PassThru kubectl "port-forward -n kubeflow $PIPELINE_API_POD 3002:8888"
    $env:ML_PIPELINE_SERVICE_PORT = 3002
    npm run mock:server 3001
} finally {
    Clean-Up
}
