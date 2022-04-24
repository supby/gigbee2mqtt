package configuration

type ConfigurationService interface {
	Update(updatedConfig Configuration) error
	GetConfiguration() Configuration
}
