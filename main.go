package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"strings"

	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
)

type ScalarNode struct {
	Name        string
	Description string
	Position    Position
}

type DiagramFormat int

const (
	Mermaid DiagramFormat = iota
	DrawIO
)

type MermaidDiagram struct {
	nodes     []string
	relations []string
}

type Position struct {
	X, Y float64
}

type ClassNode struct {
	ID       string
	Name     string
	Fields   []Field
	Position Position
	Width    float64
	Height   float64
}

type Field struct {
	Name       string
	Type       string
	IsRequired bool
}

type Relation struct {
	From     string
	To       string
	Type     string
	EdgeType string
}

type DirectiveNode struct {
	Name        string
	Description string
	Arguments   []ArgumentNode
	Locations   []string
	Position    Position
}

type ArgumentNode struct {
	Name         string
	Type         string
	IsRequired   bool
	DefaultValue string
}

type Diagram struct {
	Classes    []*ClassNode
	Relations  []Relation
	Scalars    []*ScalarNode
	Directives []*DirectiveNode
	format     DiagramFormat
	maxWidth   float64
	maxHeight  float64
}

const (
	ClassWidth        = 200
	ClassHeaderHeight = 30
	FieldHeight       = 20
	HorizontalGap     = 100
	VerticalGap       = 80
	StartX            = 50
	StartY            = 50
)

const (
	ClassStyle   = `swimlane;fontStyle=1;align=center;verticalAlign=top;childLayout=stackLayout;horizontal=1;startSize=30;horizontalStack=0;resizeParent=1;resizeParentMax=0;resizeLast=0;collapsible=1;marginBottom=0;`
	FieldStyle   = `text;strokeColor=none;fillColor=none;align=left;verticalAlign=top;spacingLeft=4;spacingRight=4;overflow=hidden;rotatable=0;points=[[0,0.5],[1,0.5]];portConstraint=eastwest;`
	EdgeStyle    = `edgeStyle=orthogonalEdgeStyle;rounded=1;orthogonalLoop=1;jettySize=auto;html=1;`
	ScalarStyle  = `ellipse;whiteSpace=wrap;html=1;aspect=fixed;fillColor=#f5f5f5;`
	ScalarWidth  = 80
	ScalarHeight = 80
)

const (
	DirectiveStyle  = `shape=hexagon;perimeter=hexagonPerimeter2;whiteSpace=wrap;html=1;fixedSize=1;fillColor=#fff2cc;strokeColor=#d6b656;`
	DirectiveWidth  = 160
	DirectiveHeight = 90
	ArgumentStyle   = `text;strokeColor=none;fillColor=none;align=left;verticalAlign=top;spacingLeft=4;spacingRight=4;overflow=hidden;rotatable=0;points=[[0,0.5],[1,0.5]];portConstraint=eastwest;`
)

type MxFile struct {
	XMLName xml.Name `xml:"mxfile"`
	Diagram MxDiagram
}

type MxDiagram struct {
	XMLName xml.Name `xml:"diagram"`
	Name    string   `xml:"name,attr"`
	Model   MxGraphModel
}

type MxGraphModel struct {
	XMLName xml.Name `xml:"mxGraphModel"`
	Root    MxRoot
}

type MxRoot struct {
	XMLName xml.Name `xml:"root"`
	Cells   []MxCell `xml:"mxCell"`
}

type MxCell struct {
	ID       string      `xml:"id,attr"`
	Value    string      `xml:"value,attr,omitempty"`
	Style    string      `xml:"style,attr,omitempty"`
	Parent   string      `xml:"parent,attr"`
	Vertex   string      `xml:"vertex,attr,omitempty"`
	Edge     string      `xml:"edge,attr,omitempty"`
	Source   string      `xml:"source,attr,omitempty"`
	Target   string      `xml:"target,attr,omitempty"`
	Geometry *MxGeometry `xml:"mxGeometry,omitempty"`
}

type MxGeometry struct {
	XMLName  xml.Name `xml:"mxGeometry"`
	X        float64  `xml:"x,attr,omitempty"`
	Y        float64  `xml:"y,attr,omitempty"`
	Width    float64  `xml:"width,attr,omitempty"`
	Height   float64  `xml:"height,attr,omitempty"`
	Relative string   `xml:"relative,attr,omitempty"`
}

func calculateLayout(d *Diagram) {
	iterations := 100
	totalNodes := len(d.Classes) + len(d.Scalars) + len(d.Directives)
	k := math.Sqrt(d.maxWidth * d.maxHeight / float64(totalNodes))

	allNodes := make([]*ClassNode, 0, totalNodes)
	allNodes = append(allNodes, d.Classes...)

	for _, scalar := range d.Scalars {
		allNodes = append(allNodes, &ClassNode{
			ID:       "scalar_" + scalar.Name,
			Name:     scalar.Name,
			Width:    ScalarWidth,
			Height:   ScalarHeight,
			Position: Position{X: rand.Float64() * d.maxWidth, Y: rand.Float64() * d.maxHeight},
		})
	}

	for _, directive := range d.Directives {
		height := DirectiveHeight + float64(len(directive.Arguments))*FieldHeight
		allNodes = append(allNodes, &ClassNode{
			ID:       "directive_" + directive.Name,
			Name:     "@" + directive.Name,
			Width:    DirectiveWidth,
			Height:   height,
			Position: Position{X: rand.Float64() * d.maxWidth, Y: rand.Float64() * d.maxHeight},
		})
	}

	for i := 0; i < iterations; i++ {
		for _, v := range allNodes {
			for _, u := range allNodes {
				if v != u {
					dx := v.Position.X - u.Position.X
					dy := v.Position.Y - u.Position.Y
					dist := math.Max(0.1, math.Sqrt(dx*dx+dy*dy))
					force := (k * k) / dist
					v.Position.X += (dx / dist) * force
					v.Position.Y += (dy / dist) * force
				}
			}
		}

		for _, rel := range d.Relations {
			var from, to *ClassNode
			for _, node := range allNodes {
				if node.Name == rel.From {
					from = node
				}
				if node.Name == rel.To {
					to = node
				}
			}
			if from != nil && to != nil {
				dx := from.Position.X - to.Position.X
				dy := from.Position.Y - to.Position.Y
				dist := math.Max(0.1, math.Sqrt(dx*dx+dy*dy))
				force := (dist * dist) / k
				dx = (dx / dist) * force
				dy = (dy / dist) * force
				from.Position.X -= dx
				from.Position.Y -= dy
				to.Position.X += dx
				to.Position.Y += dy
			}
		}
	}
}

func generateDrawIOXML(d *Diagram) []byte {
	calculateLayout(d)

	mxFile := MxFile{
		Diagram: MxDiagram{
			Name: "GraphQL Schema",
			Model: MxGraphModel{
				Root: MxRoot{
					Cells: []MxCell{
						{ID: "0"},
						{ID: "1", Parent: "0"},
					},
				},
			},
		},
	}

	for _, class := range d.Classes {
		height := ClassHeaderHeight + (float64(len(class.Fields)) * FieldHeight)

		classCell := MxCell{
			ID:     class.ID,
			Value:  class.Name,
			Style:  ClassStyle,
			Parent: "1",
			Vertex: "1",
			Geometry: &MxGeometry{
				X:      class.Position.X,
				Y:      class.Position.Y,
				Width:  ClassWidth,
				Height: height,
			},
		}
		mxFile.Diagram.Model.Root.Cells = append(mxFile.Diagram.Model.Root.Cells, classCell)

		for i, field := range class.Fields {
			fieldValue := field.Name + ": " + field.Type
			if field.IsRequired {
				fieldValue += "!"
			}

			fieldCell := MxCell{
				ID:     fmt.Sprintf("%s_f%d", class.ID, i),
				Value:  fieldValue,
				Style:  FieldStyle,
				Parent: class.ID,
				Vertex: "1",
				Geometry: &MxGeometry{
					X:      0,
					Y:      ClassHeaderHeight + float64(i)*FieldHeight,
					Width:  ClassWidth,
					Height: FieldHeight,
				},
			}
			mxFile.Diagram.Model.Root.Cells = append(mxFile.Diagram.Model.Root.Cells, fieldCell)
		}
	}

	for i, rel := range d.Relations {
		edgeCell := MxCell{
			ID:     fmt.Sprintf("e%d", i),
			Parent: "1",
			Edge:   "1",
			Source: rel.From,
			Target: rel.To,
			Style:  EdgeStyle,
			Geometry: &MxGeometry{
				Relative: "1",
			},
		}
		mxFile.Diagram.Model.Root.Cells = append(mxFile.Diagram.Model.Root.Cells, edgeCell)
	}

	for _, scalar := range d.Scalars {
		scalarNode := &ClassNode{
			ID:       "scalar_" + scalar.Name,
			Name:     scalar.Name,
			Width:    ScalarWidth,
			Height:   ScalarHeight,
			Position: Position{},
		}
		d.Classes = append(d.Classes, scalarNode)
	}

	calculateLayout(d)

	for _, scalar := range d.Scalars {
		scalarCell := MxCell{
			ID:     "scalar_" + scalar.Name,
			Value:  scalar.Name,
			Style:  ScalarStyle,
			Parent: "1",
			Vertex: "1",
			Geometry: &MxGeometry{
				X:      scalar.Position.X,
				Y:      scalar.Position.Y,
				Width:  ScalarWidth,
				Height: ScalarHeight,
			},
		}
		mxFile.Diagram.Model.Root.Cells = append(mxFile.Diagram.Model.Root.Cells, scalarCell)
	}

	for _, directive := range d.Directives {
		directiveNode := &ClassNode{
			ID:       "directive_" + directive.Name,
			Name:     "@" + directive.Name,
			Width:    DirectiveWidth,
			Height:   DirectiveHeight + float64(len(directive.Arguments))*FieldHeight,
			Position: Position{},
		}
		d.Classes = append(d.Classes, directiveNode)
	}

	calculateLayout(d)

	for _, directive := range d.Directives {
		directiveCell := MxCell{
			ID:     "directive_" + directive.Name,
			Value:  fmt.Sprintf("@%s\non %s", directive.Name, strings.Join(directive.Locations, ", ")),
			Style:  DirectiveStyle,
			Parent: "1",
			Vertex: "1",
			Geometry: &MxGeometry{
				X:      directive.Position.X,
				Y:      directive.Position.Y,
				Width:  DirectiveWidth,
				Height: DirectiveHeight + float64(len(directive.Arguments))*FieldHeight,
			},
		}
		mxFile.Diagram.Model.Root.Cells = append(mxFile.Diagram.Model.Root.Cells, directiveCell)

		for i, arg := range directive.Arguments {
			argValue := fmt.Sprintf("%s: %s", arg.Name, arg.Type)
			if arg.DefaultValue != "" {
				argValue += fmt.Sprintf(" = %s", arg.DefaultValue)
			}

			argCell := MxCell{
				ID:     fmt.Sprintf("directive_%s_arg%d", directive.Name, i),
				Value:  argValue,
				Style:  ArgumentStyle,
				Parent: "directive_" + directive.Name,
				Vertex: "1",
				Geometry: &MxGeometry{
					X:      0,
					Y:      DirectiveHeight + float64(i)*FieldHeight,
					Width:  DirectiveWidth,
					Height: FieldHeight,
				},
			}
			mxFile.Diagram.Model.Root.Cells = append(mxFile.Diagram.Model.Root.Cells, argCell)
		}
	}

	output, _ := xml.MarshalIndent(mxFile, "", "    ")
	return output
}

func isNonNullType(t ast.Type) bool {
	_, isNonNull := t.(*ast.NonNull)
	return isNonNull
}

func processSchema(doc *ast.Document, diagram *Diagram) {
	for _, def := range doc.Definitions {
		switch def := def.(type) {
		case *ast.ObjectDefinition:
			class := &ClassNode{
				ID:   "c" + def.Name.Value,
				Name: def.Name.Value,
			}

			for _, field := range def.Fields {
				class.Fields = append(class.Fields, Field{
					Name:       field.Name.Value,
					Type:       getTypeString(field.Type),
					IsRequired: isNonNullType(field.Type),
				})
			}

			diagram.Classes = append(diagram.Classes, class)

			for _, field := range def.Fields {
				if isObjectType(field.Type, doc) || isInputType(field.Type, doc) {
					diagram.Relations = append(diagram.Relations, Relation{
						From: def.Name.Value,
						To:   getBaseType(field.Type),
						Type: "uses",
					})
				}
			}
		case *ast.ScalarDefinition:
			scalar := &ScalarNode{
				Name:        def.Name.Value,
				Description: getDescription(def.Description),
			}
			diagram.Scalars = append(diagram.Scalars, scalar)
		case *ast.DirectiveDefinition:
			directive := &DirectiveNode{
				Name:        def.Name.Value,
				Description: getDescription(def.Description),
			}

			for _, loc := range def.Locations {
				directive.Locations = append(directive.Locations, loc.Value)
			}

			if def.Arguments != nil {
				for _, arg := range def.Arguments {
					argument := ArgumentNode{
						Name:       arg.Name.Value,
						Type:       getTypeString(arg.Type),
						IsRequired: isNonNullType(arg.Type),
					}

					if arg.DefaultValue != nil {
						argument.DefaultValue = formatDefaultValue(arg.DefaultValue)
					}

					directive.Arguments = append(directive.Arguments, argument)

					baseType := getBaseType(arg.Type)
					if isObjectType(arg.Type, doc) || isInputType(arg.Type, doc) || isCustomScalar(arg.Type, doc) {
						diagram.Relations = append(diagram.Relations, Relation{
							From:     "@" + directive.Name,
							To:       baseType,
							Type:     "uses",
							EdgeType: "directive",
						})
					}
				}
			}

			diagram.Directives = append(diagram.Directives, directive)
		}
	}
}

func outputDiagram(d *Diagram) {
	if d.format == DrawIO {
		output := generateDrawIOXML(d)
		fmt.Println(string(output))
	} else {
		fmt.Println("classDiagram")

		for _, directive := range d.Directives {
			fmt.Printf("class %s {\n    <<directive>>\n", directive.Name)
			for _, arg := range directive.Arguments {
				defaultValue := ""
				if arg.DefaultValue != "" {
					defaultValue = fmt.Sprintf(" = %s", arg.DefaultValue)
				}
				fmt.Printf("    +%s: %s%s\n", arg.Name, arg.Type, defaultValue)
			}
			fmt.Printf("    +on %s\n}\n", strings.Join(directive.Locations, ", "))
		}

		for _, scalar := range d.Scalars {
			fmt.Printf("class %s {\n    <<scalar>>\n}\n", scalar.Name)
		}
		for _, class := range d.Classes {
			fmt.Printf("class %s {\n", class.Name)
			for _, field := range class.Fields {
				fmt.Printf("    +%s %s\n", field.Name, field.Type)
			}
			fmt.Println("}")
		}

		for _, relation := range d.Relations {
			style := "-->"
			if relation.EdgeType == "directive" {
				style = "..>"
			}
			fmt.Printf("%s %s %s : %s\n", relation.From, style, relation.To, relation.Type)
		}
	}
}

func main() {
	// Add format flag
	formatFlag := flag.String("format", "mermaid", "Output format: mermaid or drawio")
	flag.Parse()

	// Read the GraphQL schema file
	schemaBytes, err := ioutil.ReadFile("test.graphqls")
	if err != nil {
		log.Fatalf("Error reading schema file: %v", err)
	}

	// Parse the GraphQL schema
	doc, err := parser.Parse(parser.ParseParams{
		Source: string(schemaBytes),
	})
	if err != nil {
		log.Fatalf("Error parsing schema: %v", err)
	}

	// Create new Diagram
	diagram := &Diagram{
		Classes:    make([]*ClassNode, 0),
		Relations:  make([]Relation, 0),
		Scalars:    make([]*ScalarNode, 0),
		Directives: make([]*DirectiveNode, 0),
		format:     Mermaid,
		maxWidth:   1920, // Standard screen width
		maxHeight:  1080, // Standard screen height
	}

	// Set format based on flag
	if *formatFlag == "drawio" {
		diagram.format = DrawIO
	}

	// Process the schema
	processSchema(doc, diagram)

	// Output the diagram
	outputDiagram(diagram)

}

func processInputDefinition(input *ast.InputObjectDefinition, diagram *MermaidDiagram, doc *ast.Document) {
	// Create input class definition with <<input>> stereotype
	className := input.Name.Value
	classStr := fmt.Sprintf("class %s {\n    <<input>>", className)

	// Process input fields
	for _, field := range input.Fields {
		fieldType := getTypeString(field.Type)
		classStr += fmt.Sprintf("\n    +%s %s", field.Name.Value, fieldType)

		// Add relations for object types and other input types
		if isObjectType(field.Type, doc) || isInputType(field.Type, doc) {
			relation := fmt.Sprintf("%s ..> %s : uses", className, getBaseType(field.Type))
			diagram.relations = append(diagram.relations, relation)
		}
	}

	classStr += "\n}"
	diagram.nodes = append(diagram.nodes, classStr)
}

// Modified processObjectDefinition to handle relations with input types
func processObjectDefinition(obj *ast.ObjectDefinition, diagram *MermaidDiagram, doc *ast.Document) {
	className := obj.Name.Value
	classStr := fmt.Sprintf("class %s {", className)

	for _, field := range obj.Fields {
		fieldType := getTypeString(field.Type)
		classStr += fmt.Sprintf("\n    +%s %s", field.Name.Value, fieldType)

		// Add relations for object types
		if isObjectType(field.Type, doc) {
			relation := fmt.Sprintf("%s --> %s : has", className, getBaseType(field.Type))
			diagram.relations = append(diagram.relations, relation)
		}

		// Add relations for input types in arguments
		for _, arg := range field.Arguments {
			if isInputType(arg.Type, doc) {
				relation := fmt.Sprintf("%s ..> %s : uses", className, getBaseType(arg.Type))
				diagram.relations = append(diagram.relations, relation)
			}
		}
	}

	classStr += "\n}"
	diagram.nodes = append(diagram.nodes, classStr)
}

// New function to check if a type is an input type
func isInputType(t ast.Type, doc *ast.Document) bool {
	baseType := getBaseType(t)

	// Check if the type is defined as an input type in the schema
	for _, def := range doc.Definitions {
		if input, ok := def.(*ast.InputObjectDefinition); ok {
			if input.Name.Value == baseType {
				return true
			}
		}
	}
	return false
}

func getTypeString(t ast.Type) string {
	switch t := t.(type) {
	case *ast.NonNull:
		return getTypeString(t.Type) + "!"
	case *ast.List:
		return "[" + getTypeString(t.Type) + "]"
	case *ast.Named:
		return t.Name.Value
	default:
		return "unknown"
	}
}

func getBaseType(t ast.Type) string {
	switch t := t.(type) {
	case *ast.NonNull:
		return getBaseType(t.Type)
	case *ast.List:
		return getBaseType(t.Type)
	case *ast.Named:
		return t.Name.Value
	default:
		return "unknown"
	}
}

func isObjectType(t ast.Type, doc *ast.Document) bool {
	baseType := getBaseType(t)
	return baseType != "String" && baseType != "Int" && baseType != "Float" &&
		baseType != "Boolean" && baseType != "ID" && !isInputType(t, doc)
}

func isCustomScalar(t ast.Type, doc *ast.Document) bool {
	baseType := getBaseType(t)
	if baseType == "String" || baseType == "Int" || baseType == "Float" ||
		baseType == "Boolean" || baseType == "ID" {
		return false
	}

	for _, def := range doc.Definitions {
		if scalar, ok := def.(*ast.ScalarDefinition); ok {
			if scalar.Name.Value == baseType {
				return true
			}
		}
	}
	return false
}

func getDescription(desc *ast.StringValue) string {
	if desc != nil {
		return desc.Value
	}
	return ""
}

func getDirectiveLocations(locs []string) []string {
	return locs
}

//func getDirectiveLocations(locs []ast.DirectiveLocationEnum) []string {
//	var locations []string
//	for _, loc := range locs {
//		locations = append(locations, loc.String())
//	}
//	return locations
//}

func formatDefaultValue(value ast.Value) string {
	switch v := value.(type) {
	case *ast.StringValue:
		return fmt.Sprintf(`"%s"`, v.Value)
	case *ast.IntValue:
		return v.Value
	case *ast.FloatValue:
		return v.Value
	case *ast.BooleanValue:
		return fmt.Sprintf("%t", v.Value)
	case *ast.EnumValue:
		return v.Value
	default:
		return ""
	}
}
