# Architectural Decisions

## Go-IPAM

We found out that Netbox was slow and we would have to extend Netbox to suit our needs.
Considering the findings and that but we anyhow only use a very small part of it
we decided to implement the bare minimum IMPI ourselves, see [https://github.com/metal-pod/go-ipam](https://github.com/metal-pod/go-ipam) . 

---

## Netbox (not used anymore)

In order to save time implementing software for managing IP addresses ([IPAM](https://en.wikipedia.org/wiki/IP_address_management)) and organising our datacenter infrastructure ([DCIM](https://en.wikipedia.org/wiki/Data_center_management#Data_Center_Infrastructure_Management)), we decided to take advantage of third-party software solutions.

We found that DCIM and IPAM were often combined into the same piece of software. We looked at the following offers on the market:

- [netbox](https://github.com/digitalocean/netbox)
- [phpipam](https://phpipam.net/)
- [racktables](https://www.racktables.org/)
- several smaller and older projects

After looking at the solutions, it became pretty clear that only Netbox would satisfy the expectations we had. Main advantages over the other products were:

- Versatile data model that can actually map our Metal Cloud datacenter (especially VRFs and Blade Chassis, i.e. Blade Servers can be children of a chassis and are located in machine bays)
- Complete REST API with Swagger + authentication
- Vibrant community, many Github stars
- The UI in comparison to other products was pleasing to the eye

The main issues we had on our journey with Netbox turned out to be:

- The generated Swagger Go client does not work at all (issues with type conversions, which cannot be resolved easily)
- Data model basically provides us with everything we need, but some stuff is called differently or is not there
- Performance (a lot of client calls are required in order to achieve what we want)

In order to tackle the main issue regarding the Swagger Client, we decided to introduce another microservice. The [netbox-api-proxy](https://git.f-i-ts.de/cloud-native/metal/netbox-api-proxy).

## On How Netbox is Being Utilised in Metal

The Netbox is a shadow inventory of our datacenter infrastructure and manages the IP addresses that are assigned to metal machines. We made the following decisions of what Netbox actually is being used for:

- IPAM -> Managing IP addresses (that will be assigned to the machines)
- Receive information from the `metal-api` on machines and persist them (including serial, partition and rack location, network interfaces, owners...)
- Browsing the inventory with the Netbox web UI to get an overview over the datacenter

If the Netbox is not present, machine registration, allocation and release does not function.

It is not used for:

- Querying things (like requesting all machines to find out who the owners are), the `metal-api` does not rely on data stored in the Netbox for queries
- Persisting the entire data relevant to the clients of the `metal-api` (e.g. SSH keys, OOB management credentials, ... are not stored in the Netbox). Some data is not really considered in the Netbox data model and we neither want to adopt their model nor do we want the Netbox to adopt our model.
