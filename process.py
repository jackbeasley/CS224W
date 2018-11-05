import os
import sys
from collections import defaultdict
from multiprocessing import Pool

src_hash_dir = 'mag_src_hash'
dst_hash_dir = 'mag_dst_hash'

buckets = 3000


def paper_id_to_src_file(paper_id):
    return os.path.join(src_hash_dir, '{:04d}.txt'.format(paper_id % buckets))


def paper_id_to_dst_file(paper_id):
    return os.path.join(dst_hash_dir, '{:04d}.txt'.format(paper_id % buckets))


def merge_dicts(dicts):
    """
    Given any number of dicts, shallow copy and merge into a new dict,
    precedence goes to key value pairs in latter dicts.

    From: https://stackoverflow.com/a/26853961
    """
    result = defaultdict(set)
    for dictionary in dicts:
        result.update(dictionary)
    return result


def get_in_refs_in_file((ref_filename, paper_id_set)):
    paper_to_refs = defaultdict(set)
    with open(ref_filename, 'r') as reference_file:
        for line in reference_file:
            ids = line.split()
            src_id = int(ids[0])
            dst_id = int(ids[1])

            if src_id in paper_id_set:
                paper_to_refs[src_id].add(dst_id)
    return paper_to_refs


def get_out_refs_in_file((ref_filename, paper_id_set)):
    paper_to_refs = defaultdict(set)
    with open(ref_filename, 'r') as reference_file:
        for line in reference_file:
            ids = line.split()
            src_id = int(ids[0])
            dst_id = int(ids[1])

            if dst_id in paper_id_set:
                paper_to_refs[src_id].add(dst_id)
    return paper_to_refs


def get_references(paper_ids, follow_in=True, follow_out=False, p=Pool(4)):

    results = []
    if follow_in:
        src_files_to_papers = defaultdict(set)
        for paper_id in paper_ids:
            src_files_to_papers[paper_id_to_src_file(paper_id)].add(paper_id)
        results += p.map(get_in_refs_in_file, src_files_to_papers.items())

    if follow_out:
        dst_files_to_papers = defaultdict(set)
        for paper_id in paper_ids:
            dst_files_to_papers[paper_id_to_dst_file(paper_id)].add(paper_id)
        results += p.map(get_out_refs_in_file, dst_files_to_papers.items())

    return merge_dicts(results)


def bfs(init_paper, follow_in=True, follow_out=False, levels=1):
    papers_to_references = defaultdict(list)
    pool = Pool(4)
    current_level = set([init_paper])
    for i in range(0, levels):
        paper_ref_level = get_references(current_level, follow_in, follow_out,
                                         pool)
        papers_to_references = merge_dicts(
            [papers_to_references, paper_ref_level])
        current_level = set()
        for papers in papers_to_references.values():
            current_level |= papers
    return papers_to_references


if __name__ == '__main__':
    start_paper = int(sys.argv[1])
    levels = int(sys.argv[2])
    print("Running {} level bfs starting from paper id: {}".format(
        levels, start_paper))
    results = bfs(start_paper, True, True, levels)
    print(results)
    print("found {} nodes".format(len(results.keys())))
    print("found {} edges".format(
        sum([len(neighbors) for neighbors in results.values()])))
