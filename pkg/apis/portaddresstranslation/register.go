package portaddresstranslation

const (
	// GroupName is the name of the API.
	GroupName = "k8s.deslauriers.io"

	// ConfigurationLabelKey is the label key attached to a Revision indicating by
	// which Configuration it is created.
	ConfigurationLabelKey = GroupName + "/configuration"

	// ConfigurationGenerationAnnotationKey is the annotation key attached to a Revision indicating the
	// generation of the Configuration that created this revision
	ConfigurationGenerationAnnotationKey = GroupName + "/configurationGeneration"

	// RouteLabelKey is the label key attached to a Configuration indicating by
	// which Route it is configured as traffic target.
	RouteLabelKey = GroupName + "/route"

	// RevisionLabelKey is the label key attached to k8s resources to indicate
	// which Revision triggered their creation.
	RevisionLabelKey = GroupName + "/revision"

	// RevisionUID is the label key attached to a revision to indicate
	// its unique identifier
	RevisionUID = GroupName + "/revisionUID"

	// AutoscalerLabelKey is the label key attached to a autoscaler pod indicating by
	// which Autoscaler deployment it is created.
	AutoscalerLabelKey = GroupName + "/autoscaler"

	// ServiceLabelKey is the label key attached to a Route and Configuration indicating by
	// which Service they are created.
	ServiceLabelKey = GroupName + "/service"
)
