import random
import bfs_json
import csv
import shutil
import os

def init_csv_file(results_folder, results_file):
    if not os.path.exists(results_folder):
        os.makedirs(results_folder)
        
    with open(os.path.join(results_folder, results_file), 'w') as f:
        results_writer = csv.writer(f, delimiter=',', quotechar='"', quoting=csv.QUOTE_NONNUMERIC)
        results_writer.writerow([
            "sourcePaperID",
            "followedOutLinks",
            "followedInLinks",
            "numBFSLevels",
            "outputEdgeList",
            "numFoundNodes",
            "numFoundEdges",
            "userTime",
            "systemTime",
            "totalTime",
            "maxMemory",
        ])

def write_csv_row(csv_writer, res):
    csv_writer.writerow([
            res.sourcePaperID,
            res.followedOutLinks,
            res.followedInLinks,
            res.numBFSLevels,
            res.outputEdgeList,
            res.numFoundNodes,
            res.numFoundEdges,
            res.userTime,
            res.systemTime,
            res.totalTime,
            res.maxMemory,
        ])

def check_node(node_id, levels, results_folder='crawl', results_file='crawl_results.csv'):
    with open(os.path.join(results_folder, results_file), 'a') as f:
        results_writer = csv.writer(f, delimiter=',', quotechar='"', quoting=csv.QUOTE_NONNUMERIC)
        for followOut, followIn in [(True, True), (True, False), (False, True)]:
            res = bfs_json.get_bfs_graph(node_id, levels, followOut, followIn, 20)
            write_csv_row(results_writer, res)
            shutil.move(res.outputEdgeList, os.path.join(results_folder, res.outputEdgeList))
    return


def write_ids(ids, directory='crawl'):
    with open(os.path.join(directory, 'ids.txt'), 'w') as f:
        for i in ids:
            f.write("{}\n".format(i))

def read_ids(directory='crawl'):
    with open(os.path.join(directory, 'ids.txt'), 'r') as f:
        return [int(line) for line in f]            



# ids = bfs_json.get_random_nodes()
# write_ids(ids)
ids = read_ids()

def crawl_ids(ids):
    for num_levels in [3, 4, 5, 10]: 
        for paper_id in ids:
            print("paper_id: {} for {} levels".format(paper_id, num_levels))
            check_node(paper_id, num_levels, 'crawl', 'crawl_results.csv')

crawl_ids(ids)
