# Ansible building blocks

Each `## id: title` section is one block. The description says what counts
as *using* the block: the code being written has to exercise it, not just
sit near it. Edit freely; ids are what the ledger tracks, so renaming an id
resets its schedule.

Every `.yml` and `.yaml` file reaches the classifier, Ansible or not, and it
sees only these descriptions, not this note. So each one names Ansible
explicitly, to keep other YAML (Kubernetes, CI config) from matching.

## plays: Ansible plays and playbooks
Writing an Ansible play: `hosts`, `vars`, `pre_tasks`/`tasks`/`post_tasks`, `gather_facts`, or several plays in one playbook targeting different groups.

## tasks_modules: Ansible tasks and modules
Writing Ansible tasks that call modules: a `name`, a fully qualified module like `ansible.builtin.copy`, and its arguments.

## idempotence: Ansible idempotence
Keeping Ansible tasks safe to re-run: `changed_when`, `creates`/`removes` on `command` and `shell`, or replacing a shell command with a module that reports state.

## variables: Ansible variables and precedence
`vars`, `vars_files`, `group_vars`/`host_vars`, role `defaults` vs `vars`, `set_fact`, `hostvars`, or relying on Ansible's precedence order.

## facts: Ansible facts
`ansible_facts` in decisions, `gather_subset`, `setup` with filters, cacheable `set_fact`, or custom facts under `/etc/ansible/facts.d`.

## register_results: Ansible registered results
`register` on a task and reading `.rc`, `.stdout_lines`, `.changed`, `.failed`, or `.results` from the result of a looped task.

## conditionals: Ansible when conditions
`when` on Ansible tasks, blocks, or roles: multiple conditions as a list, `is defined`, `is failed`, `is changed`, `| bool`, or conditions on facts and registered results.

## loops: Ansible loops
`loop` on an Ansible task, `loop_control` with `label`, `loop_var`, or `index_var`, looping over dicts with `dict2items`, nested loops via `subelements` or `product`, or legacy `with_*` forms.

## retries: Ansible retries
`until` with `retries` and `delay` on an Ansible task, or waiting with `wait_for` and `uri` polling until a service is ready.

## handlers: Ansible handlers
`handlers` sections, `notify`, `listen` topics, `meta: flush_handlers`, or reasoning about when handlers run and what happens after a failure.

## jinja_filters: Jinja2 expressions and filters in Ansible
Non-trivial `{{ }}` expressions in Ansible YAML: filters like `default`, `map`, `selectattr`, `combine`, `json_query`, or `regex_replace`, or `ternary` and quoting rules. A plain `{{ var }}` does not count.

## templates: Ansible templates
The `ansible.builtin.template` module with `src`, `dest`, `mode`, `validate`, or `backup`, usually notifying a handler when the rendered file changes.

## file_modules: Ansible file modules
`ansible.builtin.file`, `copy`, `lineinfile`, `blockinfile`, `replace`, `stat`, or `find`, including ownership and modes, regex anchors in `lineinfile`, and `backup`.

## packages_services: Ansible packages and services
`package`, `apt`, or `dnf` tasks (`state`, `update_cache`), or `service` and `systemd_service` tasks (`enabled`, `state`, `daemon_reload`).

## users_access: Ansible users and access
`ansible.builtin.user`, `group`, `ansible.posix.authorized_key`, or `sudoers` management, including password hashing and `update_password`.

## blocks_rescue: Ansible block, rescue, and always
Grouping Ansible tasks with `block`, handling failure in `rescue`, cleaning up in `always`, or sharing `when` and `become` across a block.

## failure_control: Ansible failure control
`failed_when`, `ignore_errors`, `ignore_unreachable`, `any_errors_fatal`, `ansible.builtin.fail`, or `ansible.builtin.assert` to validate inputs early.

## roles: Ansible roles
Structuring an Ansible role (`tasks/main.yml`, `defaults`, `handlers`, `meta/main.yml` dependencies, `meta/argument_specs.yml`), or using `roles:`, `import_role`, and `include_role`.

## imports_includes: Ansible imports and includes
`import_tasks` and `import_playbook` vs `include_tasks` and `include_vars`, static vs dynamic behavior, and how `when`, `tags`, and `loop` apply to each.

## tags: Ansible tags
`tags` on Ansible tasks, roles, or plays, the special `always` and `never` tags, or inheritance through imports vs includes.

## inventory: Ansible inventory
YAML inventories with `all`, `children`, `hosts`, and `vars`, connection variables like `ansible_host` and `ansible_user`, inventory plugins, or host patterns like `web:&prod:!canary`.

## become: Ansible privilege escalation
`become`, `become_user`, and `become_method` at play, block, or task level, or `remote_user` and file permission issues with unprivileged users.

## delegation: Ansible delegation
`delegate_to`, `run_once`, `delegate_facts`, `connection: local`, or `hosts: localhost` plays that run steps on the controller.

## rolling_updates: Ansible rolling updates
`serial`, `max_fail_percentage`, `throttle`, the `free` and `host_pinned` strategies, or drain-update-restore sequences in a play.

## async_tasks: Ansible async tasks
`async` and `poll` on an Ansible task, fire-and-forget with `poll: 0`, or checking back with `async_status`.

## vault_secrets: Ansible Vault and secrets
`ansible-vault` encrypted files or `!vault` inline values, `vault_`-prefixed variable conventions, vault IDs, or `no_log` on tasks handling secrets.

## lookups: Ansible lookups
`lookup` and `query` plugins such as `file`, `env`, `template`, `password`, or `fileglob`, including `wantlist` and controller-side evaluation.

## check_mode: Ansible check and diff mode
Making Ansible tasks behave under `--check`: `check_mode`, `diff`, `ansible_check_mode`, or read-only commands with `changed_when: false`.

## collections: Ansible collections
`requirements.yml` for roles and collections, version pins, `collections:` keywords, or `galaxy.yml` and `meta/runtime.yml` metadata.

## molecule: Molecule tests
Testing Ansible roles with Molecule: `molecule.yml` drivers and platforms, `converge.yml`, or `verify.yml` assertions.
