#Requires -RunAsAdministrator

# Dot source classes in order
$import_order = @(
  'WindowsImageDependenciesInfo.ps1',
  'WindowsImageInfo.ps1',
  'DiskImageInfo.ps1'
)

$classes_dir = Join-Path -Path $PSScriptRoot -ChildPath 'classes'
foreach ($class in $import_order) {
  $class_location = Join-Path -Path $classes_dir -ChildPath $class
  try {
    . $class_location
  } catch {
    throw "Unable to dot source $class_location"
  }
}

$public_function_path  = Join-Path -Path $PSScriptRoot -ChildPath 'public'
$private_function_path = Join-Path -Path $PSScriptRoot -ChildPath 'private'

# Dot source public/private functions
$dot_source_params = @{
  Path        = @($public_function_path, $private_function_path)
  Filter      = '*.ps1'
  Recurse     = $true
  ErrorAction = 'Stop'
}
$functions = Get-ChildItem @dot_source_params

foreach ($item_to_import in $functions) {
  try {
    . $item_to_import.FullName
  } catch {
    throw "Unable to dot source [$($item_to_import.FullName)]"
  }
}

Initialize-WindowsAssessmentAndDeploymentKit

# *-* should resolve to all public functions
Export-ModuleMember -function "*-*"