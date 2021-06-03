function Initialize-WindowsAssessmentAndDeploymentKit {
  <#
    .SYNOPSIS
    Initialize your environement to enable the use of commands from the Windows
    Assessment And Deployment Tool Kit.

    .DESCRIPTION
    Run a powershell equivilent of the DnadlSetEnv.bat command to update your
    session's path with the paths of the Windows Assessment And Deployment Tool
    Kit commands.

    .PARAMETER SupportedArchitectures
    Default list of supported architectures that script with run when found.

    .PARAMETER WindowsKitRelativeRegPath
    The relative path from the architecture specific software registry path.

    .PARAMETER KitsRootRegValueName
    The name of the property that contains the Windows Deployment and Assessment
    Kit root directory.
  #>

  [cmdletbinding()]
  Param (
    [String[]]$SupportedArchitectures = ('x86', 'AMD64'),

    [String]$WindowsKitRelativeRegPath = 'Microsoft\Windows Kits\Installed Roots',

    [String]$KitsRootRegValueName = 'KitsRoot10'
  )

  if ($env:PROCESSOR_ARCHITECTURE -notin $SupportedArchitectures) {
    throw "Not implemented for PROCESSOR_ARCHITECTURE of $($env:PROCESSOR_ARCHITECTURE)"
  }

  $get_itempropertyvalue_params = @{
    'Path' = Join-Path 'HKLM:\SOFTWARE' $WindowsKitRelativeRegPath
    'Name' = $KitsRootRegValueName
  }
  $reg_key_path_found = (Get-ItemPropertyValue @get_itempropertyvalue_params) -ne $null

  $get_itempropertyvalue_params = @{
    'Path' = Join-Path 'HKLM:\SOFTWARE\WOW6432Node' $WindowsKitRelativeRegPath
    'Name' = $KitsRootRegValueName
  }
  $wow_reg_key_path_found = (Get-ItemPropertyValue @get_itempropertyvalue_params) -ne $null

  if ($wow_reg_key_path_found) {
    $regKeyPath = Join-Path 'HKLM:\Software\Wow6432Node' $WindowsKitRelativeRegPath
  }
  elseif ($reg_key_path_found) {
    $regKeyPath = Join-Path 'HKLM:\Software' $WindowsKitRelativeRegPath
  }
  else {
    throw "KitsRoot not found, can't set common path for Deployment Tools"
  }

  $KitsRoot = Get-ItemPropertyValue -Path $regKeyPath -Name $KitsRootRegValueName

  $new_paths = @()
  $new_psmodulepaths = @()

  Write-Information 'Build the D&I Root from the queried KitsRoot'
  $DandIRoot = Join-Path $KitsRoot 'Assessment and Deployment Kit\Deployment Tools'

  Write-Information 'Construct the path to WinPE directory, architecture-independent'
  $WinPERoot = Join-Path $KitsRoot 'Assessment and Deployment Kit\Windows Preinstallation Environment'
  $null = New-Item -Path Env:WinPERoot -Value $WinPERoot -Force
  $new_paths += $WinPERoot

  Write-Information 'Constructing tools paths relevant to the current Processor Architecture'
  $architecture_root = Join-Path $DandIRoot $env:PROCESSOR_ARCHITECTURE.tolower()

  $new_paths += Join-Path $architecture_root 'DISM'
  $new_psmodulepaths += Join-Path $architecture_root 'DISM'

  $new_paths += Join-Path $architecture_root 'BCDBoot'
  $new_paths += Join-Path $architecture_root 'Imaging'

  $OSCDImgRoot  = Join-Path $architecture_root 'Oscdimg'
  $null = New-Item -Path Env:OSCDImgRoot -Value $OSCDImgRoot -Force
  $new_paths += $OSCDImgRoot

  $new_paths += Join-Path $architecture_root 'Wdsmcast'

  Write-Information 'Now do the paths that apply to all architectures...'

  Write-Information ('Note that the last one in this list should not have a',
                    'trailing semi-colon to avoid duplicate semi-colons',
                    'on the last entry when the final path is assembled.' -join ' ')

  $new_paths += Join-Path $DandIRoot 'HelpIndexer'

  Write-Information 'Set WSIM path. WSIM is X86 only and ships in architecture-independent path'
  $new_paths += Join-Path $DandIRoot 'WSIM'

  Write-Information 'Set ICDRoot. ICD is X86 only'
  $new_paths += Join-Path $KitsRoot 'Assessment and Deployment Kit\Imaging and Configuration Designer\x86'


  Write-Information 'Now build the Complete path from the various tool root folders...'

  Write-Information ('Note that each fragment above should have any required trailing',
                    'semi-colon as a delimiter so we do not put any here.' -join ' ')

  Write-Information ('Note the last one appended to NewPath should be the last one',
                    'set above in the arch. neutral section which also should not',
                    'have a trailing semi-colon.' -join ' ')

  Write-Information 'Attempting to update env:Path'
  foreach ($path in $new_paths) {
    Write-Verbose "Checking for $path in env:Path"
    if ($path -in $env:Path.split(';')) {
      Write-Verbose "Path was found in env:Path. Continuing"
      Continue
    }
    Write-Information "Attempting to add $path to env:Path"
    $env:Path = "$path;$env:Path"
  }

  Write-Information 'Attempting to update env:PSModulePath'
  foreach ($mpath in $new_psmodulepaths) {
    Write-Verbose "Checking for $mpath in env:PSModulePath"
    if ($mpath -in $env:PSModulePath.split(';')) {
      Write-Verbose 'Module Path was found in env:PSModulePath. Continuing'
      Continue
    }
    Write-Information "Attempting to add $mpath to env:PSModulePath"
    $env:PSModulePath = "$mpath;$env:PSModulePath"
  }
}