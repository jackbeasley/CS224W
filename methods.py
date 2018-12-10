import snap
from collections import defaultdict
from bfs_graph import translate, create_snap_graph, get_bfs_graph
from mag_api import mag_evaluate, mag_get_title, mag_get_date

# Search Path Count
def get_in_degrees(Graph):
    InDegV = snap.TIntPrV()
    snap.GetNodeInDegV(Graph, InDegV)
    return {item.GetVal1() : item.GetVal2() for item in InDegV}

def get_out_degrees(Graph):
    OutDegV = snap.TIntPrV()
    snap.GetNodeOutDegV(Graph, OutDegV)
    return {item.GetVal1() : item.GetVal2() for item in OutDegV}

def compute_n_minus(start, in_degrees, Graph):
    n_minus = defaultdict(int)
    in_deg = in_degrees.copy()
    while len(start) > 0:
        node = start.pop()
        if in_degrees[node] == 0:
            n_minus[node] = 1
        else:
            total = 0
            for citation in Graph.GetNI(node).GetInEdges():
                total += n_minus[citation]
            n_minus[node] = total

        for ref in Graph.GetNI(node).GetOutEdges():
            in_deg[ref] -= 1
            if in_deg[ref] == 0:
                start.add(ref)
    return n_minus

def compute_n_plus(start, out_degrees, Graph):
    n_plus = defaultdict(int)
    out_deg = out_degrees.copy()
    while len(start) > 0:
        node = start.pop()
        if out_degrees[node] == 0:
            n_plus[node] = 1
        else:
            total = 0
            for reference in Graph.GetNI(node).GetOutEdges():
                total += n_plus[reference]
            n_plus[node] = total

        for citation in Graph.GetNI(node).GetInEdges():
            out_deg[citation] -= 1
            if out_deg[citation] == 0:
                start.add(citation)
    return n_plus

def spc(Graph):
    '''Calculate SPC value for each edge in Graph.'''
    in_degrees = get_in_degrees(Graph)
    out_degrees = get_out_degrees(Graph)

    n_minus = compute_n_minus(set([node for node in in_degrees if in_degrees[node] == 0]),\
            in_degrees, Graph)
    n_plus = compute_n_plus(set([node for node in out_degrees if out_degrees[node] == 0]),\
            out_degrees, Graph)

    spc_counts = defaultdict(int)
    for edge in Graph.Edges():
        src = edge.GetSrcNId()
        dst = edge.GetDstNId()
        spc_counts[(src, dst)] = n_minus[src] * n_plus[dst]
    return spc_counts

# Edge betweenness
def edge_betweenness(Graph):
    '''Calculate edge betweenness for each edge in Graph.'''
    Nodes = snap.TIntFltH()
    Edges = snap.TIntPrFltH()
    snap.GetBetweennessCentr(Graph, Nodes, Edges, 1.0, True)
    return {(edge.GetVal1(), edge.GetVal2()): Edges[edge] for edge in Edges}

# METHODS
def mpa(start_node, translate, counts, Graph):
    '''MPA using given counts as edge weights.'''
    node = start_node
    nodes = []
    while Graph.GetNI(node).GetOutDeg() != 0:
        next_node = -1
        largest = -1
        for neighbor in Graph.GetNI(node).GetOutEdges():
            if counts[(node, neighbor)] > largest and neighbor not in nodes:
                largest = counts[(node, neighbor)]
                next_node = neighbor
        if next_node < 0:
            break
        node = next_node
        nodes.append(next_node)
    return [translate[nid] for nid in nodes]

def mpa_betweenness(start_node, translate, Graph):
    '''Run MPC using edge betweenness values.'''
    return mpa(start_node, translate, edge_betweenness(Graph), Graph)

def mpa_spc(start_node, translate, Graph):
    '''Run MPA using SPC values.'''
    return mpa(start_node, translate, spc(Graph), Graph)

def mpa_val(start_node, translate, values, Graph):
    '''Construct main path by greedily choosing node with highest value.'''
    node = start_node
    nodes = []
    while Graph.GetNI(node).GetOutDeg() != 0:
        next_node = -1
        largest = -1
        for neighbor in Graph.GetNI(node).GetOutEdges():
            if values[neighbor] > largest and neighbor not in nodes:
                largest = values[neighbor]
                next_node = neighbor
        if next_node < 0:
            break
        node = next_node
        nodes.append(next_node)
    return [translate[nid] for nid in nodes]

def validate_graph(Graph, snap_to_mag):
    '''
    Delete one of the pair for bidirectional edges in given Graph.
    '''
    bad_edges = set()
    for edge in Graph.Edges():
        src_nid = edge.GetSrcNId()
        dst_nid = edge.GetDstNId()
        if Graph.IsEdge(dst_nid, src_nid):
            first = min(src_nid, dst_nid)
            second = max(src_nid, dst_nid)
            bad_edges.add((first, second))
    for bad_edge in bad_edges:
        date1 = mag_get_date(snap_to_mag[bad_edge[0]])
        date2 = mag_get_date(snap_to_mag[bad_edge[1]])
        if date1 < date2:
            #print "Deleting edge (%d, %d)" % (snap_to_mag[bad_edge[1]], snap_to_mag[bad_edge[0]])
            Graph.DelEdge(bad_edge[1], bad_edge[0])
        else:
            #print "Deleting edge (%d, %d)" % (snap_to_mag[bad_edge[0]], snap_to_mag[bad_edge[1]])
            Graph.DelEdge(bad_edge[0], bad_edge[1])

# BASELINES
def pagerank(paper, snap_to_mag, Graph):
    PRankH = snap.TIntFltH()
    snap.GetPageRank(Graph, PRankH)
    PRankH.SortByDat(False)
    return PRankH

def node_betweenness(snap_to_mag, Graph):
    Nodes = snap.TIntFltH()
    Edges = snap.TIntPrFltH()
    snap.GetBetweennessCentr(Graph, Nodes, Edges, 1.0, True)
    return Nodes

# EVALUATION
def get_all_results(paper, levels=2, follow_out=True, follow_in=False):
    Graph, mag_to_snap = get_bfs_graph(paper, levels, follow_out, follow_in)
    snap_to_mag = {v: k for k, v in mag_to_snap.iteritems()}
    validate_graph(Graph, snap_to_mag)
    pr = pagerank(mag_to_snap[paper], snap_to_mag, Graph)
    nb = node_betweenness(snap_to_mag, Graph)
    
    start_paper = mag_to_snap[paper]
    mpa_spc_results = mpa_spc(start_paper, snap_to_mag, Graph)
    mpa_bw_results = mpa_betweenness(start_paper, snap_to_mag, Graph)
    pr_results = mpa_val(start_paper, snap_to_mag, pr, Graph)
    node_bw_results = mpa_val(start_paper, snap_to_mag, nb, Graph)

    return {"MPA (SPC)": mpa_spc_results,
            "MPA (Edge Betweenness)": mpa_bw_results,
            "MPA (PageRank)": pr_results,
            "MPA (Node Betweenness)": node_bw_results
            }, Graph