#!/usr/bin/python
import argparse
import hashlib
import logging
import os
import shutil
import subprocess
import tempfile

import jinja2


def GetGitHash():
  # The image tag is based on the githash.
  git_hash = subprocess.check_output(["git", "rev-parse", "--short", "HEAD"])
  git_hash = git_hash.strip().decode("utf-8")
  modified_files = subprocess.check_output(["git", "ls-files", "--modified"])
  untracked_files = subprocess.check_output(
      ["git", "ls-files", "--others", "--exclude-standard"])
  if modified_files or untracked_files:
    diff = subprocess.check_output(["git", "diff"])
    sha = hashlib.sha256()
    sha.update(diff)
    diffhash = sha.hexdigest()[0:7]
    git_hash = "{0}-dirty-{1}".format(git_hash, diffhash)
  return git_hash


if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  parser = argparse.ArgumentParser(
      description="Build Docker images for PyTorch samples.")

  parser.add_argument(
      "--registry",
      type=str,
      help="The docker registry to use.")

  args = parser.parse_args()

  this_file = __file__
  filename = "Dockerfile.template"
  images_dir = os.path.dirname(this_file)
  loader = jinja2.FileSystemLoader(images_dir)

  base_image = "pytorch/pytorch:latest"
  dockerfile_contents = jinja2.Environment(loader=loader).get_template(
      filename).render(base_image=base_image)
  context_dir = tempfile.mkdtemp(prefix="tmpContentxt")
  logging.info("context_dir: %s", context_dir)
  if not os.path.exists(context_dir):
    os.makedirs(context_dir)
  dockerfile = os.path.join(context_dir, 'Dockerfile')
  with open(dockerfile, 'w') as hf:
    hf.write(dockerfile_contents)

  root_dir = os.path.abspath(os.path.join(images_dir, '..', '..'))

  # List of paths to copy relative to root.
  sources = [
      "examples/pytorch_sample/pytorch_sample/pytorch_smoke.py",
  ]

  for s in sources:
    src_path = os.path.join(root_dir, s)
    dest_path = os.path.join(context_dir, os.path.basename(s))
    if os.path.exists(dest_path):
      os.unlink(dest_path)
    if os.path.isdir(src_path):
      shutil.copytree(src_path, dest_path)
    else:
      shutil.copyfile(src_path, dest_path)

  image = args.registry + "/pytorch_sample"

  image += ":" + GetGitHash()
  subprocess.check_call(["docker", "build", "-t", image, context_dir])
  logging.info("Built image: %s", image)
  subprocess.check_call(["gcloud", "docker", "--", "push", image])
  #subprocess.check_call(["docker", "push", image])
  logging.info("Pushed image: %s", image)
