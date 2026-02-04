#    _____ __________  ________ ____ ___  _________
#   /  _  \\______   \/  _____/|    |   \/   _____/
#  /  /_\  \|       _/   \  ___|    |   /\_____  \ 
# /    |    \    |   \    \_\  \    |  / /        \
# \____|__  /____|_  /\______  /______/ /_______  /
#         \/       \/        \/                 \/ 
# 
# Copyright (C) 2023 Siddharth Muralee

# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.

# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

import click
import logging
import os

import argus_components
from argus_components.common.config import parse_config, RESULTS_FOLDER
from argus_components.common.pylogger import set_global_log_level

@click.command()
@click.option("--mode", type=click.Choice(['repo', 'action', 'file']), required=True, help="The mode of operation. Choose 'repo', 'action', or 'file'.")
@click.option("--url", required=False, type=str, help="The GitHub URL. use USERNAME:TOKEN@URL for private repos.")
@click.option("--file", "file_path", required=False, type=click.Path(exists=True), help="Path to local workflow file (for file mode).")
@click.option("--output-folder", required=False, default="/tmp", help="The output folder.", type=click.Path(exists=True))
@click.option("--config", required=False, default=None, help="The config file.", type=click.Path(exists=True))
@click.option("--verbose", is_flag=True, default=False, help="Verbose mode.")
@click.option("--branch", default=None, type=str, help="The branch name.")
@click.option("--commit", default=None, type=str, help="The commit hash.")
@click.option("--tag", default=None, type=str, help="The tag.")
@click.option("--action-path", default=None, type=str, help="The (relative) path to the action.")
@click.option("--workflow-path", default=None, type=str, help="The (relative) path to the workflow.")
def main(mode, url, file_path, branch, commit, tag, output_folder, config, verbose, action_path, workflow_path):

    if verbose:
        set_global_log_level(logging.DEBUG)
    else:
        set_global_log_level(logging.INFO)

    if mode == "file":
        if not file_path:
            raise click.BadParameter("--file is required for file mode")
        if url or branch or commit or tag or action_path or workflow_path:
            raise click.BadParameter("file mode cannot be used with --url, --branch, --commit, --tag, --action-path, or --workflow-path")
    else:
        if not url:
            raise click.BadParameter("--url is required for repo and action modes")
        if file_path:
            raise click.BadParameter("--file can only be used with --mode file")

    options = [branch, commit, tag]
    options_names = ['branch', 'commit', 'tag']
    num_of_options_provided = sum(option is not None for option in options)

    if num_of_options_provided > 1:
        raise click.BadParameter("You must provide exactly one of: --branch, --commit, --tag")

    option_provided, option_value = next(((name, value) for name, value in zip(options_names, options) if value is not None), (None, None))

    if config:
        parse_config(config)

    option_dict = {
        "type": option_provided,
        "value": option_value
    } if option_provided and option_value else {}

    if mode == "repo":
        if action_path:
            raise click.BadParameter("You cannot provide --action-path in repo mode.")

        repo = argus_components.Repo(url, option_dict)
        repo.run(workflow_path)
        # repo.print_report()
        repo.save_report_to_file()
    elif mode == "action":
        if workflow_path:
            raise click.BadParameter("You cannot provide --workflow-path in action mode.")
        
        action = argus_components.Action(url, option_dict, action_path)
        action.run()
        # action.print_report()
        action.save_report_to_file()
    elif mode == "file":
        from argus_components.workflow import GHWorkflow
        from argus_components.ir import WorkflowIR
        import argus_components.taintengine as TaintEngine
        import argus_components.report as Report
        from argus_components.repo import LocalFileRepo
        
        abs_file_path = os.path.abspath(file_path)
        if not abs_file_path.endswith(('.yml', '.yaml')):
            raise click.BadParameter("File must be a .yml or .yaml file")
        
        workflow = GHWorkflow(abs_file_path, os.path.dirname(abs_file_path))
        context = LocalFileRepo(abs_file_path)
        
        ir_obj = WorkflowIR.get_IR(workflow)
        workflow_report = Report.WorkflowReport(
            TaintEngine.TaintEngine(ir_obj, context).run_workflow(),
            ir_obj
        )
        
        base_filename = os.path.splitext(os.path.basename(file_path))[0]
        output_file = RESULTS_FOLDER / f"{base_filename}.sarif"
        workflow_report.get_report(output_file)


if __name__ == "__main__":
    main()
