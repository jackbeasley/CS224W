import snap
from IPython.display import Image
from collections import defaultdict
import operator
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
        # print "looking at node", node
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
    '''
    Calculate SPC value for each edge in Graph.
    '''
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
    '''
    Calculate edge betweenness for each edge in Graph.
    '''
    Nodes = snap.TIntFltH()
    Edges = snap.TIntPrFltH()
    snap.GetBetweennessCentr(Graph, Nodes, Edges, 1.0, True)
    return {(edge.GetVal1(), edge.GetVal2()): Edges[edge] for edge in Edges}

# METHODS
def get_top_pr_nodes(pr, nodes):
    pr = dict(pr)
    pairs = sorted([(node, pr[node]) for node in nodes], key=lambda a:a[1], reverse=True)
    return [a[0] for a in pairs[:5]]

def mpa(counts, start_node, translate, pr, Graph):
    node = start_node
    results = []
    while Graph.GetNI(node).GetOutDeg() != 0:
        next_node = -1
        largest = -1
        for neighbor in Graph.GetNI(node).GetOutEdges():
            if counts[(node, neighbor)] > largest:
                largest = counts[(node, neighbor)]
                next_node = neighbor
        node = next_node
        results.append(translate[node])
    return get_top_pr_nodes(pr, results)

def mpa_betweenness(start_node, translate, pr, Graph):
    '''
    Run MPC using edge betweenness values.
    '''
    return mpa(edge_betweenness(Graph), start_node, translate, pr, Graph)

def mpa_spc(start_node, translate, pr, Graph):
    '''
    Run MPA using SPC values.
    '''
    return mpa(spc(Graph), start_node, translate, pr, Graph)


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
        Graph.DelEdge(bad_edge[0], bad_edge[1])
        Graph.DelEdge(bad_edge[1], bad_edge[0])
        #date1 = mag_get_date(snap_to_mag[bad_edge[0]])
        #date2 = mag_get_date(snap_to_mag[bad_edge[1]])
        #if date1 < date2:
        #    print "Deleting edge (%d, %d)" % (snap_to_mag[bad_edge[1]], snap_to_mag[bad_edge[0]])
        #    Graph.DelEdge(bad_edge[1], bad_edge[0])
        #else:
        #    print "Deleting edge (%d, %d)" % (snap_to_mag[bad_edge[0]], snap_to_mag[bad_edge[1]])
        #    Graph.DelEdge(bad_edge[0], bad_edge[1])

def citations(paper, snap_to_mag, Graph):
    '''
    Return papers with most citations in the Graph.
    '''
    InDegV = snap.TIntPrV()
    snap.GetNodeInDegV(Graph, InDegV)
    in_degrees = [(snap_to_mag[item.GetVal1()], item.GetVal2()) for item in InDegV\
            if snap_to_mag[item.GetVal1()] != paper]
    in_degrees.sort(key=operator.itemgetter(1), reverse=True)
    return [a[0] for a in in_degrees[:5]]

def pagerank(paper, snap_to_mag, Graph):
    PRankH = snap.TIntFltH()
    snap.GetPageRank(Graph, PRankH)
    PRankH.SortByDat(False)
    pagerank_scores = [(snap_to_mag[item], PRankH[item]) for item in PRankH\
            if snap_to_mag[item] != paper]
    return [a[0] for a in pagerank_scores[:5]], pagerank_scores

def hits(paper, snap_to_mag, Graph):
    NIdHubH = snap.TIntFltH()
    NIdAuthH = snap.TIntFltH()
    snap.GetHits(Graph, NIdHubH, NIdAuthH)
    NIdHubH.SortByDat(False)
    NIdAuthH.SortByDat(False)

    hub_scores = [(snap_to_mag[item], NIdHubH[item]) for item in NIdHubH\
            if snap_to_mag[item] != paper]
    authority_scores = [(snap_to_mag[item], NIdAuthH[item]) for item in NIdAuthH\
            if snap_to_mag[item] != paper]
    return [a[0] for a in hub_scores[:5]], [a[0] for a in authority_scores[:5]]

def node_betweenness(paper, snap_to_mag, Graph):
    Nodes = snap.TIntFltH()
    Edges = snap.TIntPrFltH()
    snap.GetBetweennessCentr(Graph, Nodes, Edges, 1.0, True)
    centralities = sorted([(snap_to_mag[node], Nodes[node]) for node in Nodes\
            if snap_to_mag[node] != paper], key=lambda a: a[1], reverse=True)
    return [a[0] for a in centralities[:5]]

def get_all_results(paper, levels=2, follow_out=True, follow_in=False):
    Graph, mag_to_snap = get_bfs_graph(paper, levels, follow_out, follow_in)
    snap_to_mag = {v: k for k, v in mag_to_snap.iteritems()}
    validate_graph(Graph, snap_to_mag)
    
    start_paper = mag_to_snap[paper]
    pr_results, pr = pagerank(start_paper, snap_to_mag, Graph)
    mpa_spc_results = mpa_spc(start_paper, snap_to_mag, pr, Graph)
    mpa_bw_results = mpa_betweenness(start_paper, snap_to_mag, pr, Graph)
    citations_results = citations(start_paper, snap_to_mag, Graph)
    hubs_results, auth_results = hits(start_paper, snap_to_mag, Graph)
    node_bw_results = node_betweenness(start_paper, snap_to_mag, Graph)

    return {"PageRank": pr_results,
            "MPA (SPC)": mpa_spc_results,
            "MPA (Edge Betweenness)": mpa_bw_results,
            "Citations": citations_results,
            "Hubs": hubs_results,
            "Authorities": auth_results,
            "Node Betweenness": node_bw_results
            }, Graph
