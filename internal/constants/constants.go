package constants

const MetaPrefix = "cat-gate.cybozu.io/"

const PodSchedulingGateName = MetaPrefix + "gate"
const CatGateImagesHashAnnotation = MetaPrefix + "images-hash"

const ImageHashAnnotationField = ".metadata.annotations.images-hash"

const LevelWarning = 1
const LevelDebug = -1
