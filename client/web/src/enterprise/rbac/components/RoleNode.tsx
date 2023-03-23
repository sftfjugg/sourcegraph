import React, { useCallback, useMemo, useState } from 'react'

import { mdiChevronUp, mdiChevronDown, mdiDelete, mdiLock } from '@mdi/js'
import { noop } from 'lodash'
import { animated } from 'react-spring'

import {
    Button,
    Icon,
    Text,
    Tooltip,
    ErrorAlert,
    Collapse,
    CollapseHeader,
    CollapsePanel,
    H4,
    useCheckboxes,
    useForm,
    Form,
    SubmissionResult,
    Alert,
    useAnimatedAlert,
} from '@sourcegraph/wildcard'

import { LoaderButton } from '../../../components/LoaderButton'
import { RoleFields } from '../../../graphql-operations'
import { prettifySystemRole } from '../../../util/settings'
import { PermissionsMap, useDeleteRole, useSetPermissions } from '../backend'

import { ConfirmDeleteRoleModal } from './ConfirmDeleteRoleModal'
import { PermissionsList } from './Permissions'

import styles from './RoleNode.module.scss'

interface RoleNodeProps {
    node: RoleFields
    refetch: () => void
    allPermissions: PermissionsMap
}

interface RoleNodePermissionsFormValues {
    permissions: string[]
}

const ModifiableRoleNode: React.FunctionComponent<RoleNodeProps> = ({ node, refetch, allPermissions }) => {
    const [isExpanded, setIsExpanded] = useState<boolean>(false)
    const [showConfirmDeleteModal, setShowConfirmDeleteModal] = useState<boolean>(false)

    const successMessage = 'Permissions successfully updated.'
    const {
        isShown: alertIsShown,
        show: showAlert,
        ref: alertRef,
        style: alertStyle,
    } = useAnimatedAlert({ autoDuration: 'short', ariaAnnouncement: { message: successMessage } })

    const handleOpenChange = (isOpen: boolean): void => {
        setIsExpanded(isOpen)
    }
    const openModal = useCallback<React.MouseEventHandler>(event => {
        event.stopPropagation()
        setShowConfirmDeleteModal(true)
    }, [])
    const closeModal = useCallback(() => {
        setShowConfirmDeleteModal(false)
    }, [])

    const [deleteRole, { loading: deleteRoleLoading, error: deleteRoleError }] = useDeleteRole(() => {
        closeModal()
        refetch()
    }, closeModal)

    const roleName = useMemo(() => (node.system ? prettifySystemRole(node.name) : node.name), [node.system, node.name])

    const onDelete = useCallback<React.FormEventHandler>(
        async event => {
            event.preventDefault()
            await deleteRole({ variables: { role: node.id } })
        },
        [deleteRole, node.id]
    )

    const { nodes: permissionNodes } = node.permissions
    const rolePermissionIDs = useMemo(() => permissionNodes.map(permission => permission.id), [permissionNodes])

    const [setPermissions, { loading: setPermissionsLoading, error: setPermissionsError }] = useSetPermissions(() => {
        refetch()
        showAlert()
    })

    const onSubmit = (values: RoleNodePermissionsFormValues): SubmissionResult => {
        // We handle any error by destructuring the query result directly
        setPermissions({ variables: { role: node.id, permissions: values.permissions } }).catch(noop)
    }
    const defaultFormValues: RoleNodePermissionsFormValues = { permissions: rolePermissionIDs }
    const { formAPI, ref, handleSubmit } = useForm({
        initialValues: defaultFormValues,
        onSubmit,
    })
    const {
        input: { isChecked, onBlur, onChange },
    } = useCheckboxes('permissions', formAPI)

    const { value } = formAPI.fields.permissions

    const isUpdateDisabled = useMemo(() => {
        // Compare which values were initially selected against the current values. We
        // will disable the button if the values are the same.
        const initialSet = new Set(rolePermissionIDs)
        const currentSet = new Set(value)
        if (initialSet.size !== currentSet.size) {
            return false
        }
        for (const item of initialSet) {
            if (!currentSet.has(item)) {
                return false
            }
        }
        return true
    }, [rolePermissionIDs, value])

    const error = deleteRoleError || setPermissionsError
    return (
        <li className={styles.roleNode}>
            {showConfirmDeleteModal && (
                <ConfirmDeleteRoleModal onCancel={closeModal} role={node} onConfirm={onDelete} />
            )}

            <Collapse isOpen={isExpanded} onOpenChange={handleOpenChange}>
                <div className="d-flex">
                    <CollapseHeader
                        as={Button}
                        className={styles.roleNodeCollapsibleHeader}
                        aria-label={isExpanded ? 'Collapse section' : 'Expand section'}
                        outline={true}
                        variant="icon"
                    >
                        <Icon
                            data-caret={true}
                            className="mr-1 bg-red"
                            aria-hidden={true}
                            svgPath={isExpanded ? mdiChevronUp : mdiChevronDown}
                        />

                        <header className="d-flex flex-column justify-content-center mr-2">
                            <div className="d-flex align-items-center">
                                <H4 className="m-0">{roleName}</H4>

                                {node.system && <SystemLabel />}
                            </div>
                            {error && <ErrorAlert className="mt-2" error={error} />}
                        </header>
                    </CollapseHeader>

                    {!node.system && (
                        <Tooltip content="Deleting a role is an irreversible action.">
                            <Button
                                aria-label="Delete"
                                onClick={openModal}
                                disabled={deleteRoleLoading}
                                variant="danger"
                                size="sm"
                                className={styles.roleNodeDeleteBtn}
                            >
                                <Icon aria-hidden={true} svgPath={mdiDelete} className={styles.roleNodeDeleteBtnIcon} />
                            </Button>
                        </Tooltip>
                    )}
                </div>

                <CollapsePanel
                    className={styles.roleNodePermissions}
                    forcedRender={false}
                    as={Form}
                    ref={ref}
                    onSubmit={handleSubmit}
                >
                    <animated.div style={alertStyle}>
                        <Alert
                            ref={alertRef}
                            variant="success"
                            className="mb-3"
                            // The alert announcement is handled by the useAnimatedAlert hook
                            aria-live="off"
                            aria-hidden={!alertIsShown}
                        >
                            {successMessage}
                        </Alert>
                    </animated.div>
                    <PermissionsList
                        allPermissions={allPermissions}
                        isChecked={isChecked}
                        onBlur={onBlur}
                        onChange={onChange}
                    />
                    <LoaderButton
                        alwaysShowLabel={true}
                        variant="primary"
                        type="submit"
                        loading={setPermissionsLoading}
                        label="Update"
                        disabled={isUpdateDisabled}
                    />
                </CollapsePanel>
            </Collapse>
        </li>
    )
}

const LockedRoleNode: React.FunctionComponent<Pick<RoleNodeProps, 'node' | 'allPermissions'>> = ({
    node,
    allPermissions,
}) => {
    const [isExpanded, setIsExpanded] = useState<boolean>(false)
    const handleOpenChange = (isOpen: boolean): void => {
        setIsExpanded(isOpen)
    }

    const roleName = useMemo(() => (node.system ? prettifySystemRole(node.name) : node.name), [node.system, node.name])

    const isChecked = useCallback(
        (value: string) => node.permissions.nodes.some(permission => permission.id === value),
        [node.permissions.nodes]
    )

    return (
        <li className={styles.roleNode}>
            <Collapse isOpen={isExpanded} onOpenChange={handleOpenChange}>
                <div className="d-flex">
                    <CollapseHeader
                        as={Button}
                        className={styles.roleNodeCollapsibleHeader}
                        aria-label={isExpanded ? 'Collapse section' : 'Expand section'}
                        outline={true}
                        variant="icon"
                    >
                        <Icon
                            data-caret={true}
                            className="mr-1 bg-red"
                            aria-hidden={true}
                            svgPath={isExpanded ? mdiChevronUp : mdiChevronDown}
                        />

                        <header className="d-flex flex-column justify-content-center mr-2">
                            <div className="d-flex align-items-center">
                                <H4 className="m-0">{roleName}</H4>
                                <SystemLabel />
                                <Tooltip content="This role is locked. Its permissions are managed by Sourcegraph and cannot be modified.">
                                    <Icon
                                        className={styles.roleNodeLockedIcon}
                                        aria-label="Locked role"
                                        svgPath={mdiLock}
                                    />
                                </Tooltip>
                            </div>
                        </header>
                    </CollapseHeader>
                </div>

                <CollapsePanel className={styles.roleNodePermissions} forcedRender={false}>
                    <PermissionsList allPermissions={allPermissions} isChecked={isChecked} disabled={true} />
                </CollapsePanel>
            </Collapse>
        </li>
    )
}

export const RoleNode: React.FunctionComponent<RoleNodeProps> = ({ node, refetch, allPermissions }) =>
    node.system && node.name === 'SITE_ADMINISTRATOR' ? (
        <LockedRoleNode node={node} allPermissions={allPermissions} />
    ) : (
        <ModifiableRoleNode node={node} refetch={refetch} allPermissions={allPermissions} />
    )

const SystemLabel: React.FunctionComponent = () => (
    <Tooltip content="System roles are predefined by Sourcegraph. They cannot be deleted.">
        <Text className={styles.roleNodeSystemText}>System</Text>
    </Tooltip>
)
