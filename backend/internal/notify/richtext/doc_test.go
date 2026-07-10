package richtext

import "testing"

func TestBuilderProducesFlatNodes(t *testing.T) {
	doc := New().Bold("Аня").Text(" обняла ").Italic("тебя").Build()
	if len(doc.Nodes) != 3 {
		t.Fatalf("want 3 nodes, got %d", len(doc.Nodes))
	}
	if doc.Nodes[0].Kind != KindBold || doc.Nodes[0].Text != "Аня" {
		t.Fatalf("node0 = %+v", doc.Nodes[0])
	}
	if doc.Nodes[1].Kind != KindPlain || doc.Nodes[1].Text != " обняла " {
		t.Fatalf("node1 = %+v", doc.Nodes[1])
	}
}

func TestBlockNodes(t *testing.T) {
	doc := New().Text("hi").Line().Quote("note").Build()
	if doc.Nodes[1].Kind != KindLine {
		t.Fatalf("want line at 1, got %v", doc.Nodes[1].Kind)
	}
	if doc.Nodes[2].Kind != KindQuote || doc.Nodes[2].Text != "note" {
		t.Fatalf("want quote at 2, got %+v", doc.Nodes[2])
	}
}
