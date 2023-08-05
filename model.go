package main

import (
	"fmt"

	"github.com/onshape-public/go-client/onshape"
	"github.com/ungerik/go3d/mat4"
	"github.com/ungerik/go3d/quaternion"
	"github.com/ungerik/go3d/vec3"
	"github.com/ungerik/go3d/vec4"
)

type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

type PartInfo struct {
	Id         string
	Name       string
	Path ElementPath
	Appearance Color
}

type AssemblyInfo struct {
	Name       string
	Path ElementPath
}

type Transform struct {
	Translation vec3.T
	Quaternion quaternion.T
}

type ElementPath struct {
	did string
	wvm string
	wvmid string
	eid string
}

type Occurrence interface {
	GetTransform() Transform
	GetId() string
}

type BaseOccurrence struct {
	Transform Transform
	Id string
}

func (b BaseOccurrence) GetTransform() Transform {
	return b.Transform
}

func (b BaseOccurrence) GetId() string {
	return b.Id
}

type AssemblyOccurrence struct {
	BaseOccurrence
	Assembly *AssemblyInfo
	Children []Occurrence
}

type PartOccurrence struct {
	BaseOccurrence
	Part *PartInfo
}

type ModelData struct {
	DocumentInfo *onshape.BTDocumentInfo
	AssemblyDef  *onshape.BTAssemblyDefinitionInfo
	PartInfoList []PartInfo
	AssemblyInfoList []AssemblyInfo
	Occurrences  []Occurrence
}

type ElementKey struct {
	did string
	wvmid string
	eid string
}

// Document/WVM/Element => Part Info List
type ElementPathToPartList map[ElementKey][]onshape.BTPartMetadataInfo

func NewTransform(m mat4.T) Transform {
	return Transform {
		Translation: vec3.T{m.Get(3, 0), m.Get(3, 1), m.Get(3, 2)},
		Quaternion: m.Quaternion(),
	}
}

func float64ArrayTo32(arr64 []float64) []float32 {
	arr32 := make([]float32, len(arr64))
	for i, v := range arr64 {
		arr32[i] = float32(v)
	}

	return arr32
}

func TransformFromArray(arr []float64) Transform {
	t := float64ArrayTo32(arr)
	return NewTransform(mat4.T{
		vec4.T{t[0], t[1], t[2], t[3]},
		vec4.T{t[4], t[5], t[6], t[7]},
		vec4.T{t[8], t[9], t[10], t[11]},
		vec4.T{t[12], t[13], t[14], t[15]},
	})
}

// Returns pointer to added child
func (a *AssemblyOccurrence) AddChild(child Occurrence) {
	a.Children = append(a.Children, child)
}

func (a *AssemblyOccurrence) GetChild(id string) Occurrence {
	for _, child := range a.Children {
		if child.GetId() == id {
			return child
		}
	}

	return nil
}

func GetAssemblyDefinitionInfo(o *Onshape) *onshape.BTAssemblyDefinitionInfo {
	assemblyDef, resp, err := o.Client.AssemblyApi. 
		GetAssemblyDefinition(
			o.Ctx,
			o.Config.BaseElement.did,
			o.Config.BaseElement.wvm,
			o.Config.BaseElement.wvmid,
			o.Config.BaseElement.eid).
		IncludeMateConnectors(true).IncludeMateFeatures(true).Execute()

	if err != nil || (resp != nil && resp.StatusCode >= 300) {
		fatalError("GetAssemblyDefinitionInfo():", err)
	}
	
	return assemblyDef
}

func GetDocumentInfo(o *Onshape) *onshape.BTDocumentInfo {
	docInfo, resp, err := o.Client.DocumentApi.GetDocument(o.Ctx, o.Config.BaseElement.did).Execute()

	if err != nil || (resp != nil && resp.StatusCode >= 300) {
		fatalError("GetDocumentInfo():", err)
	}

	return docInfo
}

func GetPartsInfo(o *Onshape, did string, wvm string, wvmid string, eid string) []onshape.BTPartMetadataInfo {
	partsInfo, resp, err := o.Client.PartApi.GetPartsWMVE(o.Ctx, did, wvm, wvmid, eid).Execute()

	if err != nil || (resp != nil && resp.StatusCode >= 300) {
		fatalError("GetPartsInfo():", err)
	}

	return partsInfo
}

func GetElementInfo(o *Onshape, did string, wvm string, wvmid string, eid string) onshape.BTDocumentElementInfo {
	elements, resp, err := o.Client.DocumentApi.GetElementsInDocument(o.Ctx, did, wvm, wvmid).ElementId(eid).Execute()

	if err != nil || (resp != nil && resp.StatusCode >= 300) {
		fatalError("GetElementInfo():", err)
	}

	return elements[0]
}

func getPartInfoList(o *Onshape, assemblyDef *onshape.BTAssemblyDefinitionInfo) []PartInfo {
	var partInfoList []PartInfo = make([]PartInfo, 0)
	
	elementPathToPartList := make(ElementPathToPartList)

	for _, part := range assemblyDef.Parts {
		if *part.IsStandardContent {
			continue
		}

		did := *part.DocumentId
		wvm := "m"
		wvmid := *part.DocumentMicroversion
		eid := *part.ElementId

		key := ElementKey{
			did: did,
			wvmid: wvmid,
			eid: eid,
		}

		if elementPathToPartList[key] == nil {
			elementPathToPartList[key] = GetPartsInfo(o, did, wvm, wvmid, eid)
		}

		partInfos := elementPathToPartList[key]

		var name string
		var appearance Color
		for _, partInfo := range partInfos {
			if *partInfo.PartId != *part.PartId {
				continue
			}

			opacity := partInfo.Appearance.Opacity
			color := partInfo.Appearance.Color
			name = *partInfo.Name
			appearance = Color{
				R: uint8(*color.Red),
				G: uint8(*color.Green),
				B: uint8(*color.Blue),
				A: uint8(*opacity),
			}
			break
		}

		partInfoList = append(partInfoList, PartInfo{
			Id: *part.PartId,
			Name: name,
			Path: ElementPath{did, wvm, wvmid, eid},
			Appearance: appearance,
		})
	}

	return partInfoList
}

func getAssemblyInfoList(o *Onshape, assemblyDef *onshape.BTAssemblyDefinitionInfo) []AssemblyInfo {
	assemblyInfoList := make([]AssemblyInfo, 0)

	for _, assembly := range assemblyDef.SubAssemblies {
		did := *assembly.DocumentId
		wvm := "m"
		wvmid := *assembly.DocumentMicroversion
		eid := *assembly.ElementId
		assemblyInfoList = append(assemblyInfoList, AssemblyInfo{
			Name: *GetElementInfo(o, did, wvm, wvmid, eid).Name,
			Path: ElementPath{did, wvm, wvmid, eid},
		})
	}

	return assemblyInfoList
}

func findPart(partInfoList []PartInfo, elementPath ElementPath, partId string) *PartInfo {
	for _, part := range partInfoList {
		if part.Path == elementPath && partId == part.Id {
			return &part
		}
	}

	panic("could not find part in part info list")
}

func findAssembly(assemblyInfoList []AssemblyInfo, elementPath ElementPath) *AssemblyInfo {
	for _, assembly := range assemblyInfoList {
		if assembly.Path == elementPath {
			return &assembly
		}
	}

	panic("count not find assembly in assembly info list")
}

func NewModelData(o *Onshape) ModelData {
	assemblyDef := GetAssemblyDefinitionInfo(o)
	
	partInfoList := getPartInfoList(o, assemblyDef)
	assemblyInfoList := getAssemblyInfoList(o, assemblyDef)

	occurrences := make(map[string]Occurrence, 0)

	// Get first-level sub assemblies and parts
	for _, occ := range assemblyDef.RootAssembly.Occurrences {
		if len(occ.Path) == 1 {
			for _, instance := range assemblyDef.RootAssembly.Instances {
				if *instance.Id == occ.Path[0] {
					if *instance.Type == onshape.BTAssemblyInstanceTypeAssembly {
						occurrences[occ.Path[0]] = &AssemblyOccurrence{
							BaseOccurrence: BaseOccurrence{Id: occ.Path[0], Transform: TransformFromArray(occ.Transform)},
							Assembly: findAssembly(assemblyInfoList, ElementPath{*instance.DocumentId, "m", *instance.DocumentMicroversion, *instance.ElementId}),
							Children: make([]Occurrence, 0),
						}
					} else if *instance.Type == onshape.BTAssemblyInstanceTypePart {
						occurrences[occ.Path[0]] = &PartOccurrence{
							BaseOccurrence: BaseOccurrence{Id: occ.Path[0], Transform: TransformFromArray(occ.Transform)},
							Part: findPart(partInfoList, ElementPath{*instance.DocumentId, "m", *instance.DocumentMicroversion, *instance.ElementId}, *instance.PartId),
						}
					}
				}
			}
		}
	}

	var addOccurrenceNode func(path []string, assembly *AssemblyOccurrence, baseOcc BaseOccurrence)
	addOccurrenceNode = func(path []string, assembly *AssemblyOccurrence, baseOcc BaseOccurrence) {
		fmt.Println(path, assembly.Assembly.Name)
		for _, sub := range assemblyDef.SubAssemblies {
			elementPath := ElementPath{*sub.DocumentId, "m", *sub.DocumentMicroversion, *sub.ElementId}
			if elementPath == assembly.Assembly.Path {
				for _, inst := range sub.Instances {
					if *inst.Id == path[0] {
						if *inst.Type == onshape.BTAssemblyInstanceTypePart {
							assembly.AddChild(PartOccurrence{
								BaseOccurrence: baseOcc,
								Part: findPart(partInfoList, ElementPath{
									did: *inst.DocumentId,
									wvm: "m",
									wvmid: *inst.DocumentMicroversion,
									eid: *inst.ElementId,
								}, *inst.PartId),
							})
						} else if *inst.Type == onshape.BTAssemblyInstanceTypeAssembly {
							assemblyChild := assembly.GetChild(path[0])
							if assemblyChild == nil {
								var assemblyBaseOcc BaseOccurrence
								for _, thisAssemblyOcc := range assemblyDef.RootAssembly.Occurrences {
									id := thisAssemblyOcc.Path[len(thisAssemblyOcc.Path) - 1] 
									if id == path[0] {
										assemblyBaseOcc = BaseOccurrence{
											Transform: TransformFromArray(thisAssemblyOcc.Transform),
											Id: id,
										}
										break
									}
								}

								assemblyOcc := &AssemblyOccurrence{
									BaseOccurrence: assemblyBaseOcc,
									Assembly: findAssembly(assemblyInfoList, ElementPath{
										did: *inst.DocumentId,
										wvm: "m",
										wvmid: *inst.DocumentMicroversion,
										eid: *inst.ElementId,
									}),
									Children: make([]Occurrence, 0),
								}
								assembly.AddChild(assemblyOcc)

								if len(path) == 1 {
									addOccurrenceNode(path, assemblyOcc, baseOcc)
								} else {
									addOccurrenceNode(path[1:], assemblyOcc, baseOcc)
								}
							} else {
								// If it does contain the sub assembly just get the assembly and recurse
								assemblyChildCast, ok := assemblyChild.(*AssemblyOccurrence)
								if ok {
									if len(path) == 1 {
										addOccurrenceNode(path, assemblyChildCast, baseOcc)
									} else {
										addOccurrenceNode(path[1:], assemblyChildCast, baseOcc)
									}
								}
							}
						}
						break
					}
				}
			}
		}
	}

	for _, occ := range assemblyDef.RootAssembly.Occurrences {
		if len(occ.Path) > 1 {
			assembly, ok := occurrences[occ.Path[0]].(*AssemblyOccurrence)
			if ok {
				addOccurrenceNode(occ.Path[1:], assembly, BaseOccurrence{
					Transform: TransformFromArray(occ.Transform),
					Id: occ.Path[len(occ.Path) - 1],
				})
			}
		}
	}

	// Ignore ids and get occurrences only
	occurrenceList := make([]Occurrence, 0) 
	for _, v := range occurrences {
		occurrenceList = append(occurrenceList, v)
	}

	return ModelData{
		DocumentInfo: GetDocumentInfo(o),
		AssemblyDef: assemblyDef,
		PartInfoList: partInfoList,
		AssemblyInfoList: assemblyInfoList,
		Occurrences: occurrenceList,
	}
}
