import snap
import psutil
import subprocess
import json
import random
import time


def translate(mag_to_snap, mag_id, Graph):
    """Translate MAG IDs to SNAP IDs."""
    if mag_id not in mag_to_snap:
        mag_to_snap[mag_id] = Graph.AddNode()
    return mag_to_snap[mag_id]


def create_snap_graph(edgelist_path, save_path):
    """Load MAG edge list into SNAP graph."""
    Graph = snap.TNGraph.New()
    mag_to_snap = {}
    with open(edgelist_path, 'r') as edgelist:
        for edge in edgelist:
            ids = edge.split()
            src_id = translate(mag_to_snap, long(ids[0]), Graph)
            dst_id = translate(mag_to_snap, long(ids[1]), Graph)
            # Switch edge ordering if want b->a if a cites b
            # Right now, has edge a->b if a cites b
            if not Graph.IsEdge(src_id, dst_id):
                Graph.AddEdge(src_id, dst_id)

    # Save SNAP graph to binary file
    FOut = snap.TFOut(save_path)
    Graph.Save(FOut)
    FOut.Flush()

    return Graph, mag_to_snap


def get_bfs_graph(paper_id, levels, follow_out, follow_in, fdLimit=100):
    """
    Wrapper around Go program in bfs.go

    :param paper_id: the int64 MAG paper id to start the BFS from
    :param levels: how many steps to BFS away from the origin paper
    :param follow_out: Follow out links (i.e. if source paper a cites paper b, follow the edge a -> b to paper b)
    :param follow_in: Follow in links (i.e. if source paper a is cited by paper b, follow the edge b -> a to paper b)
    :param fdLimit:
    :returns: a snap graph of the papers found in the BFS along with a map that maps MAG ids (int64) to SNAP ids (int32)
    """
    cmd = ["go", "run", "bfs.go"]
    if follow_out:
        cmd.append('-out')
    if follow_in:
        cmd.append('-in')
    cmd += ['-levels', str(levels)]
    cmd += ['-fdLimit', str(fdLimit)]
    cmd.append(str(paper_id))
    p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    out, err = p.communicate()

    return json.loads(out)


MAG_MIN_ID = 37
MAG_MAX_ID = 2895778921


def get_random_nodes(n=1000):
    cmd = 'gshuf -n {} PaperReferences.txt'.format(n)
    print cmd

    p = psutil.Popen(
        cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, shell=True)

    while p.poll() is None:
        print p.name()
        print p.cpu_times()
        print p.memory_info()
        print p.ppid()
        print p.status()
        time.sleep(2)

    out, err = p.communicate()
    p.wait()

    return set([int(line[0:line.find('\t')]) for line in out.splitlines()])
