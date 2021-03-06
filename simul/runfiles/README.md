# RUNFILES for simulations

Each simulation can have one or more .toml-files that describe a number of experiments
to be run on localhost or deterlab.

The .toml-files are split in two parts, separated by an empty line. The first
part consists of one or more 'global' variables that describe all experiments.
 
The second part starts with a line of variables that have to be defined for each
experiment, where each experiment makes up one line.

## Necessary variables

- Simulation - what simulation to run
- Hosts - how many hosts to instantiate
- Servers - how many servers to use

## sda.SimulationBFTree

If you use the `sda.SimulationBFTree`, the following variables are also available:

- BF - branching factor: how many children each node has
- Depth - the depth of the tree in levels below the root-node
- Rounds - for how many rounds the simulation should run

## Timeouts

Two timeout variables are available:

- RunWait - how many seconds to wait for a run (one line of .toml-file) to finish
    (default: 180)
- ExperimentWait - how many seconds to wait for the while experiment to finish
    (default: RunWait * #Runs)

## Experimental

- SingleHost - which will reduce the tree to use only one host per server, and
thus speeding up connections again

# Example

    Simulation = "ExampleHandlers"
    Servers = 16
    BF = 2
    Rounds = 10
    #SingleHost = true

    Hosts
    3
    7
    15
    31

This will run the `ExampleHandlers`-simulation on 16 servers with a branching
factor of 2 and 10 rounds. The `SingleHost`-argument is commented out, so it
will use as many hosts as described.

In the second part, 4 experiments are defined, each only changing the number
of `Hosts`. First 3, then 7, 15, and finally 31 hosts are run on the 16
servers. For each experiment 10 rounds are started.