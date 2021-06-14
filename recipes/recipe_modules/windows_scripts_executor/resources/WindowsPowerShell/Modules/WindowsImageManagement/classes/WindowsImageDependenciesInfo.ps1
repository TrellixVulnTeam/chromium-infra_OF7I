class WindowsImageDependenciesInfo {
  # Properties
  [System.IO.FileInfo]$SourceLocation

  [System.IO.FileInfo]$DestinationLocation

  # Constructors
  WindowsImageDependenciesInfo (){}

  WindowsImageDependenciesInfo (
    $Info
  ){
    $this.SourceLocation      = $Info.Source
    $this.DestinationLocation = $Info.Destination
  }

  WindowsImageDependenciesInfo (
    [String]$Source,
    [String]$Destination
  ){
    $this.SourceLocation      = $Source
    $this.DestinationLocation = $Destination
  }
}